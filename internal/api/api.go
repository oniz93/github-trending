package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/database"
)

func NewServer(redisClient *redis.Client, db *database.PostgresConnection) *gin.Engine {
	router := gin.Default()

	router.GET("/retrieveList", handleRetrieveList(redisClient, db))
	router.POST("/trackOpenRepository", handleTrackOpenRepository(db))

	return router
}

func handleRetrieveList(redisClient *redis.Client, db *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("sessionId")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		languages := strings.Split(c.Query("languages"), ",")
		tags := strings.Split(c.Query("tags"), ",")

		repoIDs, err := db.GetTrendingRepositoryIDs(languages, tags)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve repository list"})
			return
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

		var newRepoIDs []int64
		for _, id := range repoIDs {
			if !seenRepoIDMap[id] {
				newRepoIDs = append(newRepoIDs, id)
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
		if start > len(newRepoIDs) {
			newRepoIDs = []int64{}
		} else if end > len(newRepoIDs) {
			newRepoIDs = newRepoIDs[start:]
		} else {
			newRepoIDs = newRepoIDs[start:end]
		}

		// Add the new repositories to the seen set in Redis
		if len(newRepoIDs) > 0 {
			var repoIDStrs []string
			for _, id := range newRepoIDs {
				repoIDStrs = append(repoIDStrs, fmt.Sprint(id))
			}
			redisClient.SAdd(context.Background(), sessionID, repoIDStrs)
		}

		c.JSON(http.StatusOK, gin.H{
			"sessionId":    sessionID,
			"repositories": newRepoIDs,
		})
	}
}

func handleTrackOpenRepository(db *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody struct {
			SessionID    string `json:"sessionId"`
			RepositoryID int64  `json:"repositoryId"`
		}

		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := db.TrackRepositoryView(requestBody.SessionID, requestBody.RepositoryID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track repository view"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}
