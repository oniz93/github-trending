package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	pgConnection, err := database.NewPostgresConnection(cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgConnection.DB.Close()

	milvusConnection, err := database.NewMilvusConnection(cfg.MilvusHost, cfg.MilvusPort)
	if err != nil {
		log.Fatalf("Failed to connect to Milvus: %v", err)
	}

	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Parse LAST_UPDATE_CUT duration
	lastUpdateCutDuration, err := time.ParseDuration(cfg.LastUpdateCut)
	if err != nil {
		log.Fatalf("Invalid LAST_UPDATE_CUT duration: %v", err)
	}

	// Parse SIMILARITY_LIST_SIZE
	similarityListSize, err := strconv.Atoi(cfg.SimilarityListSize)
	if err != nil {
		log.Fatalf("Invalid SIMILARITY_LIST_SIZE: %v", err)
	}

	// Run the similarity calculation periodically
	ticker := time.NewTicker(6 * time.Hour) // Run every 6 hours
	defer ticker.Stop()

	log.Println("Similarity Engine service started. Running similarity calculation periodically.")

	for range ticker.C {
		log.Println("Starting similarity calculation...")
		// Step 3.2: Build the Similarity List Generator
		// 1. Fetch all repository IDs from PostgreSQL that have been updated within the LAST_UPDATE_CUT period.
		repoIDs, err := pgConnection.GetRepositoryIDsUpdatedSince(time.Now().Add(-lastUpdateCutDuration))
		if err != nil {
			log.Printf("Failed to get repository IDs from Postgres: %v", err)
			continue
		}

		for _, repoID := range repoIDs {
			// 2. For each repository_id, query Milvus to get its embedding vector.
			embedding, err := milvusConnection.GetEmbedding(repoID)
			if err != nil {
				log.Printf("Failed to get embedding for repo %d from Milvus: %v", repoID, err)
				continue
			}

			// 3. Perform a similarity search in Milvus using that vector to find the top N nearest neighbors.
			searchResults, err := milvusConnection.SearchEmbeddings(embedding, similarityListSize)
			if err != nil {
				log.Printf("Failed to search embeddings for repo %d in Milvus: %v", repoID, err)
				continue
			}

			// 4. Connect to Redis and store this result in a Sorted Set.
			redisKey := fmt.Sprintf("similar:%d", repoID)
			var zMembers []*redis.Z
			for _, result := range searchResults {
				// Assuming the search result contains the repo_id and distance/score
				// You might need to adjust this based on the actual Milvus search result structure
				// For now, let's assume the first output field is repo_id and the distance is the score
				for _, field := range result.Fields {
					if field.Name == "repo_id" {
						repoIDFromMilvus := field.GetAsInt64(0)
						zMembers = append(zMembers, &redis.Z{Score: float64(result.Scores[0]), Member: repoIDFromMilvus})
					}
				}
			}

			_, err = redisClient.ZAdd(context.Background(), redisKey, zMembers...).Result()
			if err != nil {
				log.Printf("Failed to store similarity list for repo %d in Redis: %v", repoID, err)
				continue
			}
			log.Printf("Successfully calculated and stored similarity for repo ID: %d", repoID)
		}
		log.Println("Similarity calculation completed.")
	}
}
