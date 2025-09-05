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
	router.GET("/getReadme", handleGetReadme(cfg, minioConnection))

	return router
}

func errorResponse(c *gin.Context, code int, genericMessage string, err error, debug bool) {
	response := gin.H{"error": genericMessage}
	if debug && err != nil {
		response["debug_error"] = err.Error()
	}
	c.JSON(code, response)
}

func handleRetrieveList(cfg *config.Config, redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("sessionId")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		languages := strings.Split(c.Query("languages"), ",")
		if len(languages) == 1 && languages[0] == "" {
			languages = []string{}
		}
		tags := strings.Split(c.Query("tags"), ",")
		if len(tags) == 1 && tags[0] == "" {
			tags = []string{}
		}

		topics := strings.Split(c.Query("topics"), ",")
		if len(topics) == 1 && topics[0] == "" {
			topics = []string{}
		}

		var recommendedRepoIDs []int64

		// 1. Query the repository_views table for user history
		userHistoryRepoIDs, err := pgdb.GetRecentClickedRepositoryIDs(sessionID, 15) // Get last 15 clicked repos
		if err != nil {
			log.Printf("Failed to get user history from Postgres: %v", err)
			// Fallback to generic trending if history fails
			trendingRepoIDs, err := chdb.GetTrendingRepositoryIDsByGrowth(30) // Using 7 days as default
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "Failed to retrieve trending repository list from ClickHouse", err, cfg.Debug)
				return
			}
			// If filters are provided, filter the list of IDs
			if len(languages) > 0 || len(tags) > 0 || len(topics) > 0 {
				filteredRepoIDs, err := pgdb.FilterRepositoryIDs(trendingRepoIDs, languages, tags, topics)
				if err != nil {
					errorResponse(c, http.StatusInternalServerError, "Failed to filter repository list", err, cfg.Debug)
					return
				}
				recommendedRepoIDs = filteredRepoIDs
			} else {
				recommendedRepoIDs = trendingRepoIDs
			}
		} else if len(userHistoryRepoIDs) == 0 {
			// 2. If no user history exists, fall back to generic trending based on ClickHouse growth
			trendingRepoIDs, err := chdb.GetTrendingRepositoryIDsByGrowth(30) // Using 7 days as default
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "Failed to retrieve trending repository list from ClickHouse", err, cfg.Debug)
				return
			}

			// If filters are provided, filter the list of IDs
			if len(languages) > 0 || len(tags) > 0 || len(topics) > 0 {
				filteredRepoIDs, err := pgdb.FilterRepositoryIDs(trendingRepoIDs, languages, tags, topics)
				if err != nil {
					errorResponse(c, http.StatusInternalServerError, "Failed to filter repository list", err, cfg.Debug)
					return
				}
				recommendedRepoIDs = filteredRepoIDs
			} else {
				recommendedRepoIDs = trendingRepoIDs
			}
		} else {
			// 3. If user history exists, aggregate similar repositories
			candidateScores := make(map[int64]float64)
			for _, historyRepoID := range userHistoryRepoIDs {
				redisKey := fmt.Sprintf("similar:%d", historyRepoID)

				// Try to get from Redis first
				similarRepos, err := redisClient.ZRevRangeWithScores(context.Background(), redisKey, 0, 49).Result()
				if err != nil || len(similarRepos) == 0 {
					// If not in Redis, get from PostgreSQL
					jsonData, err := pgdb.GetRepositorySimilarity(historyRepoID)
					if err != nil {
						log.Printf("Failed to get similar repos from Postgres for %d: %v", historyRepoID, err)
						continue
					}

					if jsonData != nil {
						var zMembers []redis.Z
						if err := json.Unmarshal(jsonData, &zMembers); err != nil {
							log.Printf("Failed to unmarshal similarity data for repo %d: %v", historyRepoID, err)
							continue
						}

						// Add to Redis with a 24-hour TTL
						_, err = redisClient.ZAdd(context.Background(), redisKey, zMembers...).Result()
						if err != nil {
							log.Printf("Failed to store similarity list for repo %d in Redis: %v", historyRepoID, err)
						}
						redisClient.Expire(context.Background(), redisKey, 24*time.Hour)

						similarRepos, _ = redisClient.ZRevRangeWithScores(context.Background(), redisKey, 0, 49).Result()
					}
				}

				for _, z := range similarRepos {
					repoID, _ := strconv.ParseInt(fmt.Sprintf("%.0f", z.Member), 10, 64)
					candidateScores[repoID] += z.Score
				}
			}

			// Convert map to slice for sorting
			type scoredRepo struct {
				ID    int64
				Score float64
			}
			var scoredRepos []scoredRepo
			for id, score := range candidateScores {
				scoredRepos = append(scoredRepos, scoredRepo{ID: id, Score: score})
			}

			// Sort by score in descending order
			sort.Slice(scoredRepos, func(i, j int) bool {
				return scoredRepos[i].Score > scoredRepos[j].Score
			})

			for _, sr := range scoredRepos {
				recommendedRepoIDs = append(recommendedRepoIDs, sr.ID)
			}
		}

		// Filter out repositories that have already been seen in this session
		seenRepoIDs, err := redisClient.SMembers(context.Background(), sessionID).Result()
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to retrieve seen repositories", err, cfg.Debug)
			return
		}

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

		// Paginate the results
		page := 0
		if pageStr := c.Query("page"); pageStr != "" {
			page, _ = strconv.Atoi(pageStr)
		}
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
		log.Printf("Final repo IDs after pagination: %d", len(finalRepoIDs))

		// Fetch full repository data
		repositories, err := pgdb.GetRepositoriesDataByIDs(finalRepoIDs)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "Failed to retrieve full repository data", err, cfg.Debug)
			return
		}

		// Create a new response structure to control the output
		type RepositoryResponse struct {
			Repository          models.Repository      `json:"repository"`
			Owner               models.Owner           `json:"owner"`
			Stats               *models.RepositoryStat `json:"stats,omitempty"`
			TotalViews          int                    `json:"total_views"`
			RecentViews         int                    `json:"recent_views"`
			TrendingScore       float64                `json:"trending_score"`
			IsTrending          bool                   `json:"is_trending"`
			LastCrawledAt       time.Time              `json:"last_crawled_at"`
			LastUpdatedAt       time.Time              `json:"last_updated_at"`
			Crawled             bool                   `json:"crawled"`
			ProgrammingLanguage string                 `json:"programming_language"`
			SpokenLanguage      string                 `json:"spoken_language"`
			PopularityScore     float64                `json:"popularity_score"`
			GrowthScore         float64                `json:"growth_score"`
			ProjectHealth       float64                `json:"project_health"`
			SimilarityScore     float64                `json:"similarity_score"`
		}

		responseRepos := make([]RepositoryResponse, len(repositories))

		// Add stats from ClickHouse and filter fields
		for i, repo := range repositories {
			stats, err := chdb.GetRepositoryStats(int64(repo.Repository.ID))
			if err != nil {
				log.Printf("Failed to get stats for repository %d from ClickHouse: %v", repo.Repository.ID, err)
			}

			var lastStat *models.RepositoryStat
			if len(stats) > 0 {
				lastStat = &stats[len(stats)-1]
			}

			responseRepos[i] = RepositoryResponse{
				Repository:          repo.Repository,
				Owner:               repo.Owner,
				Stats:               lastStat,
				TotalViews:          repo.TotalViews,
				RecentViews:         repo.RecentViews,
				TrendingScore:       repo.TrendingScore,
				IsTrending:          repo.IsTrending,
				LastCrawledAt:       repo.LastCrawledAt,
				LastUpdatedAt:       repo.LastUpdatedAt,
				Crawled:             repo.Crawled,
				ProgrammingLanguage: repo.ProgrammingLanguage,
				SpokenLanguage:      repo.SpokenLanguage,
				PopularityScore:     repo.PopularityScore,
				GrowthScore:         repo.GrowthScore,
				ProjectHealth:       repo.ProjectHealth,
				SimilarityScore:     repo.SimilarityScore,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"sessionId":    sessionID,
			"repositories": responseRepos,
		})
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

func handleGetReadme(cfg *config.Config, minioConnection *database.MinioConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoID := c.Query("repoId")
		if repoID == "" {
			errorResponse(c, http.StatusBadRequest, "repoId query parameter is required", nil, cfg.Debug)
			return
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

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(result["html"]))
	}
}
