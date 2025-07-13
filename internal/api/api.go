package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/database"
)

func NewServer(redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) *gin.Engine {
	router := gin.Default()

	// Add CORS middleware
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	router.GET("/retrieveList", handleRetrieveList(redisClient, pgdb, chdb))
	router.POST("/trackOpenRepository", handleTrackOpenRepository(pgdb))

	return router
}

func handleRetrieveList(redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) gin.HandlerFunc {
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

		var recommendedRepoIDs []int64

		// 1. Query the repository_views table for user history
		userHistoryRepoIDs, err := pgdb.GetRecentClickedRepositoryIDs(sessionID, 15) // Get last 15 clicked repos
		if err != nil {
			log.Printf("Failed to get user history from Postgres: %v", err)
			// Fallback to generic trending if history fails
			repoIDs, err := pgdb.GetTrendingRepositoryIDs(languages, tags)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve repository list"})
				return
			}
			recommendedRepoIDs = repoIDs
		} else if len(userHistoryRepoIDs) == 0 {
			// 2. If no user history exists, fall back to generic trending
			repoIDs, err := pgdb.GetTrendingRepositoryIDs(languages, tags)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve repository list"})
				return
			}
			recommendedRepoIDs = repoIDs
		} else {
			// 3. If user history exists, aggregate similar repositories
			candidateScores := make(map[int64]float64)
			for _, historyRepoID := range userHistoryRepoIDs {
				redisKey := fmt.Sprintf("similar:%d", historyRepoID)
				// Fetch top 50 similar repos for each history item
				similarRepos, err := redisClient.ZRevRangeWithScores(context.Background(), redisKey, 0, 49).Result()
				if err != nil {
					log.Printf("Failed to get similar repos from Redis for %d: %v", historyRepoID, err)
					continue
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve seen repositories"})
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

		// Add the new repositories to the seen set in Redis
		if len(finalRepoIDs) > 0 {
			var repoIDStrs []string
			for _, id := range finalRepoIDs {
				repoIDStrs = append(repoIDStrs, fmt.Sprint(id))
			}
			redisClient.SAdd(context.Background(), sessionID, repoIDStrs)
		}

		// Fetch full repository data
		repositories, err := pgdb.GetRepositoriesDataByIDs(finalRepoIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve full repository data"})
			return
		}

		// Add stats from ClickHouse
		for i, repo := range repositories {
			stats, err := chdb.GetRepositoryStats(int64(repo.Repository.ID))
			if err != nil {
				log.Printf("Failed to get stats for repository %d from ClickHouse: %v", repo.Repository.ID, err)
			}
			repositories[i].Stats = stats
		}

		c.JSON(http.StatusOK, gin.H{
			"sessionId":    sessionID,
			"repositories": repositories,
		})
	}
}

func handleTrackOpenRepository(pgdb *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody struct {
			SessionID    string `json:"sessionId"`
			RepositoryID int64  `json:"repositoryId"`
		}

		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := pgdb.TrackRepositoryView(requestBody.SessionID, requestBody.RepositoryID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track repository view"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}
