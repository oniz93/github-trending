package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	pgConnection, err := database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgConnection.DB.Close()

	qdrantConnection, err := database.NewQdrantConnection("qdrant_db", 6333)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer qdrantConnection.Close()

	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Run the similarity calculation periodically
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	defer ticker.Stop()

	log.Println("Similarity Engine service started. Running similarity calculation periodically.")

	for range ticker.C {
		log.Println("Starting similarity calculation...")
		// Step 3.2: Build the Similarity List Generator
		// 1. Fetch all repository IDs from PostgreSQL that have been updated within the LAST_UPDATE_CUT period.
		repoIDs, err := pgConnection.GetRepositoryIDsUpdatedSince(time.Now().Add(-cfg.LastUpdateCut))
		if err != nil {
			log.Printf("Failed to get repository IDs from Postgres: %v", err)
			continue
		}

		for _, repoID := range repoIDs {
			// 2. For each repository_id, query Qdrant to get its embedding vector.
			// Qdrant doesn't have a direct "GetEmbedding" by ID. We need to search for it.
			// This is a simplified approach, in a real scenario, you might store the embedding
			// directly in Postgres or have a dedicated service to retrieve it.
			// For now, we'll search for the point itself.
			searchRes, err := qdrantConnection.Search(context.Background(), "repositories", []float32{}, 1) // Search for the point itself
			if err != nil || len(searchRes) == 0 || searchRes[0].GetId().GetNum() != uint64(repoID) {
				log.Printf("Failed to get embedding for repo %d from Qdrant: %v", repoID, err)
				continue
			}
			embedding := searchRes[0].GetVectors().GetVector().GetData()

			// 3. Perform a similarity search in Qdrant using that vector to find the top N nearest neighbors.
			searchResults, err := qdrantConnection.Search(context.Background(), "repositories", embedding, uint64(cfg.SimilarityListSize))
			if err != nil {
				log.Printf("Failed to search embeddings for repo %d in Qdrant: %v", repoID, err)
				continue
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
					continue
				}
			}
			log.Printf("Successfully calculated and stored similarity for repo ID: %d", repoID)
		}
		log.Println("Similarity calculation completed.")
	}
}
