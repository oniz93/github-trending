package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
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
	"github.com/golang/snappy"
	"github.com/google/uuid"
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
	router.GET("/repository/:id", handleGetRepositoryDetails(cfg, redisClient, pgdb, chdb))
	router.GET("/api/og", handleGenerateOGImage(cfg))

	return router
}

func handleGenerateOGImage(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoID := c.Query("id")
		if repoID == "" {
			errorResponse(c, http.StatusBadRequest, "id query parameter is required", nil, cfg.Debug)
			return
		}

		ogImageServiceURL := "http://og-image-service:3000"

		resp, err := http.Get(fmt.Sprintf("%s/?id=%s", ogImageServiceURL, repoID))
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to call OG image service", err, cfg.Debug)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errorResponse(c, http.StatusInternalServerError, fmt.Sprintf("OG image service returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes)), nil, cfg.Debug)
			return
		}

		c.Header("Content-Type", "image/png")
		c.Header("Cache-Control", "public, max-age=86400") // Cache for 1 day
		io.Copy(c.Writer, resp.Body)
	}
}

func handleGetRepositoryDetails(cfg *config.Config, redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoIDStr := c.Param("id")
		repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "Invalid repository ID", err, cfg.Debug)
			return
		}

		sessionID := c.Query("sessionId")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		// Track the view first
		if err := pgdb.TrackRepositoryView(sessionID, repoID); err != nil {
			// Log the error but don't block the user
			log.Printf("Failed to track repository view for repo %d: %v", repoID, err)
		}
		// Add the repository to the seen set in Redis
		redisClient.SAdd(context.Background(), sessionID, fmt.Sprint(repoID))

		repoData, err := pgdb.GetRepositoryByID(repoID)
		if err != nil {
			if err == sql.ErrNoRows {
				errorResponse(c, http.StatusNotFound, "Repository not found", err, cfg.Debug)
			} else {
				errorResponse(c, http.StatusInternalServerError, "Failed to retrieve repository data", err, cfg.Debug)
			}
			return
		}

		latestStat, err := chdb.GetLatestRepositoryStat(repoID)
		if err != nil {
			// Log the error but don't block the user, stats are not critical
			log.Printf("Failed to get latest stats for repo %d: %v", repoID, err)
		}

		type RepositoryDetailResponse struct {
			SessionID  string                 `json:"sessionId"`
			Repository models.Repository      `json:"repository"`
			Owner      models.Owner           `json:"owner"`
			Stats      *models.RepositoryStat `json:"stats,omitempty"`
		}

		response := RepositoryDetailResponse{
			SessionID:  sessionID,
			Repository: repoData,
			Owner:      repoData.Owner,
			Stats:      latestStat,
		}

		c.JSON(http.StatusOK, response)
	}
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

// decodeStatCache decodes a cached repository-stats payload and returns the
// most recent observation. It returns nil for missing, empty, or malformed
// entries so callers can fall back to fetching from ClickHouse.
func decodeStatCache(res interface{}) *models.RepositoryStat {
	if res == nil {
		return nil
	}
	str, ok := res.(string)
	if !ok {
		return nil
	}
	decompressed, err := decompress([]byte(str))
	if err != nil {
		return nil
	}
	var stats []models.RepositoryStat
	if err := json.Unmarshal(decompressed, &stats); err != nil || len(stats) == 0 {
		return nil
	}
	// Stats are stored newest-first (ORDER BY event_time DESC), so the first
	// element is the most recent observation.
	return &stats[0]
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
					redisClient.Set(c.Request.Context(), trendingCacheKey, compress(jsonBytes), 24*time.Hour)
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
					redisClient.Set(c.Request.Context(), filterCacheKey, compress(jsonBytes), 24*time.Hour)
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

		// Collect the repository IDs once so stats can be fetched in bulk.
		repoIDs := make([]int64, len(repositories))
		for i, repo := range repositories {
			repoIDs[i] = int64(repo.Repository.ID)
		}

		cacheKeys := make([]string, len(repoIDs))
		for i, id := range repoIDs {
			cacheKeys[i] = fmt.Sprintf("repo_stats:%d", id)
		}

		statsByRepoID := make(map[int64]*models.RepositoryStat, len(repoIDs))

		// 1. Serve whatever we can from Redis with a single MGET round-trip.
		missedIDs := make([]int64, 0, len(repoIDs))
		if cached, err := redisClient.MGet(c.Request.Context(), cacheKeys...).Result(); err == nil {
			for i, res := range cached {
				if stat := decodeStatCache(res); stat != nil {
					statsByRepoID[repoIDs[i]] = stat
					continue
				}
				missedIDs = append(missedIDs, repoIDs[i])
			}
		} else {
			missedIDs = append(missedIDs, repoIDs...)
		}

		// 2. Fetch the remaining stats from ClickHouse in one batch query and
		// backfill the Redis cache.
		if len(missedIDs) > 0 {
			batchStats, err := chdb.GetLatestRepositoryStatsBatch(missedIDs)
			if err != nil {
				log.Printf("Failed to get latest stats for %d repositories from ClickHouse: %v", len(missedIDs), err)
			}
			for _, id := range missedIDs {
				stat, ok := batchStats[id]
				if !ok {
					continue
				}
				statsByRepoID[id] = stat
				jsonBytes, err := json.Marshal([]models.RepositoryStat{*stat})
				if err == nil {
					redisClient.Set(c.Request.Context(), fmt.Sprintf("repo_stats:%d", id), compress(jsonBytes), 12*time.Hour)
				}
			}
		}

		responseRepos := make([]RepositoryResponse, len(repositories))
		for i, repo := range repositories {
			responseRepos[i] = RepositoryResponse{
				Repository: repo.Repository,
				Owner:      repo.Owner,
				Stats:      statsByRepoID[int64(repo.Repository.ID)],
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
			redisClient.Set(c.Request.Context(), buildCacheKey(sessionID), compressedResponse, 24*time.Hour)
			// If it was a new session, also cache for the empty session ID to serve subsequent initial requests
			if isNewSession {
				redisClient.Set(c.Request.Context(), cacheKey, compressedResponse, 24*time.Hour)
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
