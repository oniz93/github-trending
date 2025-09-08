package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"crypto/sha256"
	"github.com/golang/snappy"
	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/models"
)

func NewServer(cfg *config.Config, redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection, minioConnection *database.MinioConnection) *gin.Engine {
	router := gin.Default()

	// Add CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	router.Use(cors.New(corsConfig))

	router.GET("/retrieveList", handleRetrieveList(cfg, redisClient, pgdb, chdb))
	router.POST("/trackOpenRepository", handleTrackOpenRepository(cfg, redisClient, pgdb))
	router.GET("/getReadme", handleGetReadme(cfg, redisClient, minioConnection))

	return router
}

func errorResponse(c *gin.Context, code int, genericMessage string, err error, debug bool) {
	response := gin.H{"error": genericMessage}
	if debug && err != nil {
		response["debug_error"] = err.Error()
	}
	c.JSON(code, response)
}

func compress(data []byte) []byte {
	return snappy.Encode(nil, data)
}

func decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

func handleRetrieveList(cfg *config.Config, redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		originalSessionID := c.Query("sessionId")
		isNewSession := originalSessionID == ""
		sessionID := originalSessionID
		if isNewSession {
			sessionID = uuid.New().String()
		}

		languages := strings.Split(c.Query("languages"), ",")
		if len(languages) == 1 && languages[0] == "" {
			languages = []string{}
		}
		sort.Strings(languages)

		tags := strings.Split(c.Query("tags"), ",")
		if len(tags) == 1 && tags[0] == "" {
			tags = []string{}
		}
		sort.Strings(tags)

		topics := strings.Split(c.Query("topics"), ",")
		if len(topics) == 1 && topics[0] == "" {
			topics = []string{}
		}
		sort.Strings(topics)

		pageStr := c.Query("page")
		page, _ := strconv.Atoi(pageStr)

		// --- Overall Response Cache Check ---
		buildCacheKey := func(sID string) string {
			keyBuilder := strings.Builder{}
			keyBuilder.WriteString(fmt.Sprintf("retrieveList:%s:%s:%s:%s:%s", sID, strings.Join(languages, ","), strings.Join(tags, ","), strings.Join(topics, ","), pageStr))
			return fmt.Sprintf("cache:%x", sha256.Sum256([]byte(keyBuilder.String())))
		}

		cacheKey := buildCacheKey(originalSessionID)
		cachedResponse, err := redisClient.Get(c.Request.Context(), cacheKey).Result()
		if err == nil {
			decompressed, err := decompress([]byte(cachedResponse))
			if err == nil {
				var response gin.H
				if err := json.Unmarshal(decompressed, &response); err == nil {
					if isNewSession {
						response["sessionId"] = sessionID
					}
					c.JSON(http.StatusOK, response)
					return
				}
			}
		}
		// --- End of Cache Check ---

		var recommendedRepoIDs []int64

		// 1. Query the repository_views table for user history
		userHistoryRepoIDs, err := pgdb.GetRecentClickedRepositoryIDs(sessionID, 15)
		if err != nil {
			log.Printf("Failed to get user history from Postgres: %v", err)
		}

		if len(userHistoryRepoIDs) == 0 {
			// --- Generic Trending Logic ---
			trendingCacheKey := "trending_repo_ids_by_growth:30"
			cachedTrending, err := redisClient.Get(c.Request.Context(), trendingCacheKey).Result()
			if err == nil {
				decompressed, err := decompress([]byte(cachedTrending))
				if err == nil {
					json.Unmarshal(decompressed, &recommendedRepoIDs)
				}
			}

			if len(recommendedRepoIDs) == 0 {
				trendingRepoIDs, err := chdb.GetTrendingRepositoryIDsByGrowth(30)
				if err != nil {
					errorResponse(c, http.StatusInternalServerError, "Failed to retrieve trending repository list from ClickHouse", err, cfg.Debug)
					return
				}
				recommendedRepoIDs = trendingRepoIDs
				jsonBytes, err := json.Marshal(recommendedRepoIDs)
				if err == nil {
					redisClient.Set(c.Request.Context(), trendingCacheKey, compress(jsonBytes), 10*time.Minute)
				}
			}
		} else {
			// --- Personalized Recommendation Logic ---
			candidateScores := make(map[int64]float64)
			for _, historyRepoID := range userHistoryRepoIDs {
				similarRepos, err := pgdb.GetRepositorySimilarity(historyRepoID)
				if err != nil {
					log.Printf("Failed to get similar repos for %d: %v", historyRepoID, err)
					continue
				}

				var zMembers []redis.Z
				if err := json.Unmarshal(similarRepos, &zMembers); err != nil {
					log.Printf("Failed to unmarshal similarity data for repo %d: %v", historyRepoID, err)
					continue
				}

				for _, z := range zMembers {
					repoID, _ := strconv.ParseInt(fmt.Sprintf("%.0f", z.Member), 10, 64)
					candidateScores[repoID] += z.Score
				}
			}

			type scoredRepo struct {
				ID    int64
				Score float64
			}
			var scoredRepos []scoredRepo
			for id, score := range candidateScores {
				scoredRepos = append(scoredRepos, scoredRepo{ID: id, Score: score})
			}
			sort.Slice(scoredRepos, func(i, j int) bool {
				return scoredRepos[i].Score > scoredRepos[j].Score
			})
			for _, sr := range scoredRepos {
				recommendedRepoIDs = append(recommendedRepoIDs, sr.ID)
			}
		}

		// Filter if needed
		if len(languages) > 0 || len(tags) > 0 || len(topics) > 0 {
			keyBuilder := strings.Builder{}
			for _, id := range recommendedRepoIDs {
				keyBuilder.WriteString(strconv.FormatInt(id, 10))
			}
			filterCacheKey := fmt.Sprintf("filtered_ids:%x:%s:%s:%s", sha256.Sum256([]byte(keyBuilder.String())), strings.Join(languages, ","), strings.Join(tags, ","), strings.Join(topics, ","))

			cachedFiltered, err := redisClient.Get(c.Request.Context(), filterCacheKey).Result()
			if err == nil {
				decompressed, err := decompress([]byte(cachedFiltered))
				if err == nil {
					json.Unmarshal(decompressed, &recommendedRepoIDs)
				}
			} else {
				filteredRepoIDs, err := pgdb.FilterRepositoryIDs(recommendedRepoIDs, languages, tags, topics)
				if err != nil {
					errorResponse(c, http.StatusInternalServerError, "Failed to filter repository list", err, cfg.Debug)
					return
				}
				recommendedRepoIDs = filteredRepoIDs
				jsonBytes, err := json.Marshal(recommendedRepoIDs)
				if err == nil {
					redisClient.Set(c.Request.Context(), filterCacheKey, compress(jsonBytes), 10*time.Minute)
				}
			}
		}

		// Filter out seen repos
		seenRepoIDs, _ := redisClient.SMembers(context.Background(), sessionID).Result()
		seenRepoIDMap := make(map[int64]bool)
		for _, idStr := range seenRepoIDs {
			id, _ := strconv.ParseInt(idStr, 10, 64)
			seenRepoIDMap[id] = true
		}

		var finalRepoIDs []int64
		for _, id := range recommendedRepoIDs {
			if !seenRepoIDMap[id] {
				finalRepoIDs = append(finalRepoIDs, id)
			}
		}

		// Paginate
		pageSize := 50
		start := page * pageSize
		end := start + pageSize
		if start > len(finalRepoIDs) {
			finalRepoIDs = []int64{}
		} else if end > len(finalRepoIDs) {
			finalRepoIDs = finalRepoIDs[start:]
		} else {
			finalRepoIDs = finalRepoIDs[start:end]
		}

		repositories, err := pgdb.GetRepositoriesDataByIDs(finalRepoIDs)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to retrieve full repository data", err, cfg.Debug)
			return
		}

		type RepositoryResponse struct {
			Repository models.Repository      `json:"repository"`
			Owner      models.Owner           `json:"owner"`
			Stats      *models.RepositoryStat `json:"stats,omitempty"`
		}

		responseRepos := make([]RepositoryResponse, len(repositories))
		for i, repo := range repositories {
			var lastStat *models.RepositoryStat
			statsCacheKey := fmt.Sprintf("repo_stats:%d", repo.Repository.ID)
			cachedStats, err := redisClient.Get(c.Request.Context(), statsCacheKey).Result()
			if err == nil {
				decompressed, err := decompress([]byte(cachedStats))
				if err == nil {
					var stats []models.RepositoryStat
					if json.Unmarshal(decompressed, &stats) == nil && len(stats) > 0 {
						lastStat = &stats[len(stats)-1]
					}
				}
			} else {
				stats, err := chdb.GetRepositoryStats(int64(repo.Repository.ID))
				if err != nil {
					log.Printf("Failed to get stats for repository %d from ClickHouse: %v", repo.Repository.ID, err)
				} else {
					if len(stats) > 0 {
						lastStat = &stats[len(stats)-1]
					}
					jsonBytes, err := json.Marshal(stats)
					if err == nil {
						redisClient.Set(c.Request.Context(), statsCacheKey, compress(jsonBytes), 1*time.Hour)
					}
				}
			}

			responseRepos[i] = RepositoryResponse{
				Repository: repo.Repository,
				Owner:      repo.Owner,
				Stats:      lastStat,
			}
		}

		// --- Cache Final Response ---
		finalResponse := gin.H{
			"sessionId":    sessionID,
			"repositories": responseRepos,
		}
		jsonResponse, err := json.Marshal(finalResponse)
		if err == nil {
			compressedResponse := compress(jsonResponse)
			// Cache for the actual session ID (new or existing)
			redisClient.Set(c.Request.Context(), buildCacheKey(sessionID), compressedResponse, 5*time.Minute)
			// If it was a new session, also cache for the empty session ID to serve subsequent initial requests
			if isNewSession {
				redisClient.Set(c.Request.Context(), cacheKey, compressedResponse, 5*time.Minute)
			}
		}

		c.JSON(http.StatusOK, finalResponse)
	}
}

func handleTrackOpenRepository(cfg *config.Config, redisClient *redis.Client, pgdb *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody struct {
			SessionID    string `json:"sessionId"`
			RepositoryID int64  `json:"repositoryId"`
		}

		if err := c.BindJSON(&requestBody); err != nil {
			errorResponse(c, http.StatusBadRequest, "Invalid request body", err, cfg.Debug)
			return
		}

		if err := pgdb.TrackRepositoryView(requestBody.SessionID, requestBody.RepositoryID); err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to track repository view", err, cfg.Debug)
			return
		}

		// Add the repository to the seen set in Redis
		redisClient.SAdd(context.Background(), requestBody.SessionID, fmt.Sprint(requestBody.RepositoryID))

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}

func handleGetReadme(cfg *config.Config, redisClient *redis.Client, minioConnection *database.MinioConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoID := c.Query("repoId")
		if repoID == "" {
			errorResponse(c, http.StatusBadRequest, "repoId query parameter is required", nil, cfg.Debug)
			return
		}

		// --- Cache Check ---
		cacheKey := fmt.Sprintf("readme_html:%s", repoID)
		cachedHTML, err := redisClient.Get(c.Request.Context(), cacheKey).Result()
		if err == nil {
			decompressed, err := decompress([]byte(cachedHTML))
			if err == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", decompressed)
				return
			}
		}

		// Construct the object name for MinIO
		objectName := fmt.Sprintf("readmes/%s.md", repoID)

		// Get the README content from MinIO
		readmeContent, err := minioConnection.GetFile(context.Background(), objectName)
		if err != nil {
			errorResponse(c, http.StatusNotFound, "README not found", err, cfg.Debug)
			return
		}

		// Call the markup service to convert Markdown to HTML
		reqBody, err := json.Marshal(map[string]string{
			"text": string(readmeContent),
		})
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to create request body for markup service", err, cfg.Debug)
			return
		}

		resp, err := http.Post("http://markup-service:80/markup", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to call markup service", err, cfg.Debug)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Markup service returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes)), nil, cfg.Debug)
			return
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to decode markup service response", err, cfg.Debug)
			return
		}

		htmlContent := []byte(result["html"])

		// --- Cache Result ---
		compressedHTML := compress(htmlContent)
		err = redisClient.Set(c.Request.Context(), cacheKey, compressedHTML, 24*time.Hour).Err()
		if err != nil {
			log.Printf("Failed to cache README for repo %s: %v", repoID, err)
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", htmlContent)
	}
}
