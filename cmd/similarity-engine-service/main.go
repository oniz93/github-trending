package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
	"log"
	"sync"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
)

const numWorkers = 10

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	mqConnection, err := messaging.NewConnection(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer mqConnection.Close()

	pgConnection, err := database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgConnection.DB.Close()

	var qdrantConnection *database.QdrantConnection
	for i := 0; i < 5; i++ {
		qdrantConnection, err = database.NewQdrantConnection("qdrant_db", 6334)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Qdrant, retrying in 5 seconds: %v", err)
		time.Sleep(5 * time.Second)
	}
	if qdrantConnection == nil {
		log.Fatalf("Failed to connect to Qdrant after multiple retries")
	}
	defer qdrantConnection.Close()

	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Run the similarity calculation once on startup
	log.Println("Similarity Engine service started. Running initial similarity calculation.")
	calculateSimilarity(pgConnection, qdrantConnection, redisClient, cfg, mqConnection)

	// Run the similarity calculation periodically
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	defer ticker.Stop()

	log.Println("Initial similarity calculation completed. Running similarity calculation periodically.")

	for range ticker.C {
		calculateSimilarity(pgConnection, qdrantConnection, redisClient, cfg, mqConnection)
	}
}

func calculateSimilarity(pgConnection *database.PostgresConnection, qdrantConnection *database.QdrantConnection, redisClient *redis.Client, cfg *config.Config, mqConnection messaging.MQConnection) {
	log.Println("Starting similarity calculation...")
	// 1. Fetch all repository IDs from PostgreSQL that have been updated within the LAST_UPDATE_CUT period.
	repoIDs, err := pgConnection.GetRepositoryIDsUpdatedSince(time.Now().Add(-cfg.LastUpdateCut))
	if err != nil {
		log.Printf("Failed to get repository IDs from Postgres: %v", err)
		return
	}

	jobs := make(chan int64, len(repoIDs))
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, &wg, jobs, qdrantConnection, redisClient, cfg, pgConnection, mqConnection)
	}

	// Send jobs
	for _, repoID := range repoIDs {
		jobs <- repoID
	}
	close(jobs)

	wg.Wait()
	log.Println("Similarity calculation completed.")
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan int64, qdrantConnection *database.QdrantConnection, redisClient *redis.Client, cfg *config.Config, pgConnection *database.PostgresConnection, mqConnection messaging.MQConnection) {
	defer wg.Done()
	for repoID := range jobs {
		log.Printf("Worker %d started job for repo ID %d", id, repoID)
		processRepository(repoID, qdrantConnection, redisClient, cfg, pgConnection, mqConnection)
		log.Printf("Worker %d finished job for repo ID %d", id, repoID)
	}
}

func processRepository(repoID int64, qdrantConnection *database.QdrantConnection, redisClient *redis.Client, cfg *config.Config, pgConnection *database.PostgresConnection, mqConnection messaging.MQConnection) {
	// 2. For each repository_id, query Qdrant to get its embedding vector.
	searchRes, err := qdrantConnection.Search(context.Background(), "repositories", []float32{}, 1) // Search for the point itself
	if err != nil || len(searchRes) == 0 || searchRes[0].GetId().GetNum() != uint64(repoID) {
		log.Printf("Failed to get embedding for repo %d from Qdrant: %v", repoID, err)
		// If the embedding is not found, send a message to the readme_to_embed queue
		embedMsg := models.ReadmeEmbedMessage{
			RepositoryID: repoID,
			MinioPath:    fmt.Sprintf("readmes/%d.md", repoID),
		}
		embedMsgJSON, err := json.Marshal(embedMsg)
		if err != nil {
			log.Printf("Failed to marshal embed message for %d: %v", repoID, err)
		} else {
			err = mqConnection.Publish("readme_to_embed", embedMsgJSON)
			if err != nil {
				log.Printf("Failed to publish embed message for %d: %v", repoID, err)
			}
		}
		return
	}
	embedding := searchRes[0].GetVectors().GetVector().GetData()

	// 3. Perform a similarity search in Qdrant using that vector to find the top N nearest neighbors.
	searchResults, err := qdrantConnection.Search(context.Background(), "repositories", embedding, uint64(cfg.SimilarityListSize))
	if err != nil {
		log.Printf("Failed to search embeddings for repo %d in Qdrant: %v", repoID, err)
		return
	}

	// 4. Connect to Redis and store this result in a Sorted Set.
	redisKey := fmt.Sprintf("similar:%d", repoID)
	var zMembers []redis.Z
	for _, result := range searchResults {
		zMembers = append(zMembers, redis.Z{Score: float64(result.GetScore()), Member: result.GetId().GetNum()})
	}

	if len(zMembers) > 0 {
		_, err = redisClient.ZAdd(context.Background(), redisKey, zMembers...).Result()
		if err != nil {
			log.Printf("Failed to store similarity list for repo %d in Redis: %v", repoID, err)
			return
		}
	}
	log.Printf("Successfully calculated and stored similarity for repo ID: %d", repoID)
}
