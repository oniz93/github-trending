package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

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

	pgConnection, err := database.NewPostgresConnection(cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgConnection.DB.Close()

	minioConnection, err := database.NewMinioConnection(cfg.MinioEndpoint, cfg.MinioRootUser, cfg.MinioRootPassword)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	milvusConnection, err := database.NewMilvusConnection(cfg.MilvusHost, cfg.MilvusPort)
	if err != nil {
		log.Fatalf("Failed to connect to Milvus: %v", err)
	}

	readmeEmbedQueueName := "readme_to_embed"

	msgs, err := mqConnection.Consume(readmeEmbedQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming from queue %s: %v", readmeEmbedQueueName, err)
	}

	log.Printf("Embedding service started. Waiting for messages on queue: %s", readmeEmbedQueueName)

	for d := range msgs {
		var embedMsg models.ReadmeEmbedMessage
		if err := json.Unmarshal(d.Body, &embedMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			d.Ack(false)
			continue
		}

		log.Printf("Received message to embed README for repository ID: %d", embedMsg.RepositoryID)

		// 1. Query PostgreSQL for repository metadata (pushed_at)
		repo, err := pgConnection.GetRepositoryByID(embedMsg.RepositoryID)
		if err != nil {
			log.Printf("Failed to get repository by ID %d from Postgres: %v", embedMsg.RepositoryID, err)
			d.Nack(false, true) // Requeue for retry
			continue
		}

		// 2. Check LAST_UPDATE_CUT
		lastUpdateCutDuration, err := time.ParseDuration(cfg.LastUpdateCut)
		if err != nil {
			log.Printf("Invalid LAST_UPDATE_CUT duration: %v", err)
			d.Ack(false) // Acknowledge and skip if config is bad
			continue
		}

		if repo.PushedAt.Before(time.Now().Add(-lastUpdateCutDuration)) {
			log.Printf("Skipping embedding for repo %d (pushed_at: %s) as it's older than %s", embedMsg.RepositoryID, repo.PushedAt, cfg.LastUpdateCut)
			d.Ack(false)
			continue
		}

		// 3. Fetch README content from MinIO
		readmeContent, err := minioConnection.GetFile(context.Background(), embedMsg.MinioPath)
		if err != nil {
			log.Printf("Failed to get README from MinIO for %s: %v", embedMsg.MinioPath, err)
			// 4. Fallback: Download README directly if not in MinIO
			if repo.ReadmeURL != "" {
				log.Printf("Attempting to download README from %s", repo.ReadmeURL)
				resp, httpErr := http.Get(repo.ReadmeURL)
				if httpErr != nil || resp.StatusCode != http.StatusOK {
					log.Printf("Failed to download README from URL %s: %v", repo.ReadmeURL, httpErr)
					d.Nack(false, true)
					continue
				}
				defer resp.Body.Close()
				readmeContent, err = io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Failed to read downloaded README content: %v", err)
					d.Nack(false, true)
					continue
				}
			} else {
				log.Printf("No README URL available for repo %d, skipping embedding.", embedMsg.RepositoryID)
				d.Ack(false)
				continue
			}
		}

		// 5. Generate embedding (Placeholder - actual model integration needed)
		// For now, create a dummy embedding
		embedding := generateEmbedding(readmeContent)

		// 6. Save embedding to Milvus
		if err := milvusConnection.InsertEmbedding(embedMsg.RepositoryID, embedding); err != nil {
			log.Printf("Failed to insert embedding for repo %d into Milvus: %v", embedMsg.RepositoryID, err)
			d.Nack(false, true) // Requeue for retry
			continue
		}

		log.Printf("Successfully processed and embedded README for repository ID: %d", embedMsg.RepositoryID)
		d.Ack(false)
	}
}

// generateEmbedding is a placeholder function for generating embeddings.
// In a real application, this would involve calling an ML model.
func generateEmbedding(content []byte) []float32 {
	// Dummy implementation: create a 384-dimensional vector with dummy values
	embedding := make([]float32, 384)
	for i := range embedding {
		embedding[i] = float32(i) / 384.0 // Dummy values
	}
	return embedding
}