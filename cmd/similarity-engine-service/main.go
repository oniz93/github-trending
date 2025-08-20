package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	numWorkers = 10
	maxRetries = 5
	retryDelay = 5 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var chConnection *database.ClickHouseConnection
	for i := 0; i < maxRetries; i++ {
		chConnection, err = database.NewClickHouseConnection(cfg.ClickHouseHost, cfg.ClickHousePort, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.ClickHouseDB)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to ClickHouse: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse after %d retries: %v", maxRetries, err)
	}
	defer chConnection.Close()

	var mqConnection messaging.MQConnection
	for i := 0; i < maxRetries; i++ {
		mqConnection, err = messaging.NewConnection(cfg.RabbitMQURL)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d retries: %v", maxRetries, err)
	}
	defer mqConnection.Close()

	var pgConnection *database.PostgresConnection
	for i := 0; i < maxRetries; i++ {
		pgConnection, err = database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to PostgreSQL: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL after %d retries: %v", maxRetries, err)
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

	var redisClient *redis.Client
	for i := 0; i < maxRetries; i++ {
		redisClient, err = database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Redis: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to Redis after %d retries: %v", maxRetries, err)
	}
	pgConnection.WithRedis(redisClient)

	// Run the similarity calculation once on startup
	log.Println("Similarity Engine service started. Running initial similarity calculation.")
	calculateSimilarity(pgConnection, qdrantConnection, redisClient, cfg, mqConnection, chConnection)

	// Run the similarity calculation periodically
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	defer ticker.Stop()

	log.Println("Initial similarity calculation completed. Running similarity calculation periodically.")

	for range ticker.C {
		calculateSimilarity(pgConnection, qdrantConnection, redisClient, cfg, mqConnection, chConnection)
	}
}

func calculateSimilarity(pgConnection *database.PostgresConnection, qdrantConnection *database.QdrantConnection, redisClient *redis.Client, cfg *config.Config, mqConnection messaging.MQConnection, chConnection *database.ClickHouseConnection) {
	log.Println("Starting similarity calculation...")
	// 1. Fetch all repository IDs from ClickHouse that have been updated within the LAST_UPDATE_CUT period.
	repoIDs, err := chConnection.GetRepositoryIDsToUpdate(time.Now().Add(-cfg.LastUpdateCut))
	if err != nil {
		log.Printf("Failed to get repository IDs from ClickHouse: %v", err)
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
	vectors, err := qdrantConnection.GetVectors(context.Background(), "repositories", []uint64{uint64(repoID)})
	if err != nil || len(vectors) == 0 {
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
	embedding := vectors[0].GetVectors().GetVector().GetData()

	// 3. Perform a similarity search in Qdrant using that vector to find the top N nearest neighbors.
	searchResults, err := qdrantConnection.Search(context.Background(), "repositories", embedding, uint64(cfg.SimilarityListSize), uint64(repoID))
	if err != nil {
		log.Printf("Failed to search embeddings for repo %d in Qdrant: %v", repoID, err)
		return
	}

	// 4. Get source repo topics and languages
	sourceRepo, err := pgConnection.GetRepositoryByID(repoID)
	if err != nil {
		log.Printf("Failed to get source repo %d from Postgres: %v", repoID, err)
		return
	}

	// 5. Calculate new similarity scores
	var newScores []redis.Z
	for _, result := range searchResults {
		var finalScore float64
		candidateRepoID := int64(result.GetId().GetNum())

		// Get candidate repo topics and languages
		candidateRepo, err := pgConnection.GetRepositoryByID(candidateRepoID)
		if err != nil {
			log.Printf("Failed to get candidate repo %d from Postgres: %v", candidateRepoID, err)
			continue
		}

		// Calculate topic similarity
		topicScore := jaccardSimilarity(sourceRepo.Topics, candidateRepo.Topics)

		// Calculate language similarity
		var sourceLanguages []string
		for lang := range sourceRepo.Languages {
			sourceLanguages = append(sourceLanguages, lang)
		}
		var candidateLanguages []string
		for lang := range candidateRepo.Languages {
			candidateLanguages = append(candidateLanguages, lang)
		}
		languageScore := jaccardSimilarity(sourceLanguages, candidateLanguages)

		// Combine scores
		finalScore = (0.6 * float64(result.GetScore())) + (0.3 * topicScore) + (0.1 * languageScore)

		newScores = append(newScores, redis.Z{Score: finalScore, Member: candidateRepoID})
	}

	// 6. Connect to Redis and store this result in a Sorted Set.
	redisKey := fmt.Sprintf("similar:%d", repoID)

	// 7. Store the similarity data in PostgreSQL.
	jsonData, err := json.Marshal(newScores)
	if err != nil {
		log.Printf("Failed to marshal similarity data for repo %d: %v", repoID, err)
		return
	}
	err = pgConnection.UpsertRepositorySimilarity(repoID, jsonData)
	if err != nil {
		log.Printf("Failed to store similarity data for repo %d in PostgreSQL: %v", repoID, err)
		return
	}

	// 8. If the key exists in Redis, update it without touching the TTL.
	exists, err := redisClient.Exists(context.Background(), redisKey).Result()
	if err != nil {
		log.Printf("Failed to check if key exists in Redis for repo %d: %v", repoID, err)
		return
	}

	if exists == 1 {
		if len(newScores) > 0 {
			_, err = redisClient.ZAdd(context.Background(), redisKey, newScores...).Result()
			if err != nil {
				log.Printf("Failed to store similarity list for repo %d in Redis: %v", repoID, err)
				return
			}
		}
	}

	log.Printf("Successfully calculated and stored similarity for repo ID: %d", repoID)
}

func jaccardSimilarity(a, b []string) float64 {
	setA := make(map[string]bool)
	for _, item := range a {
		setA[item] = true
	}

	setB := make(map[string]bool)
	for _, item := range b {
		setB[item] = true
	}

	intersection := 0
	for item := range setA {
		if setB[item] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
