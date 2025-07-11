package main

import (
	"context"
	"encoding/json"
	qdrant_go_client "github.com/qdrant/go-client/qdrant"
	"github.com/streadway/amqp"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const numWorkers = 100

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

	minioConnection, err := database.NewMinioConnection(cfg.MinioEndpoint, cfg.MinioRootUser, cfg.MinioRootPassword)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	qdrantConnection, err := database.NewQdrantConnection("qdrant_db", 6334)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer qdrantConnection.Close()

	const vectorSize = 384
	log.Printf("Ensuring Qdrant collection 'repositories' exists with vector size %d", vectorSize)
	err = qdrantConnection.CreateCollection(context.Background(), "repositories", vectorSize)
	if err != nil {
		log.Fatalf("Failed to create or verify Qdrant collection: %v", err)
	}

	readmeEmbedQueueName := "readme_to_embed"

	msgs, err := mqConnection.Consume(readmeEmbedQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming from queue %s: %v", readmeEmbedQueueName, err)
	}

	var wg sync.WaitGroup
	jobs := make(chan amqp.Delivery, 100)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg, jobs, pgConnection, minioConnection, qdrantConnection)
	}

	log.Printf("Embedding service started. Waiting for messages on queue: %s", readmeEmbedQueueName)

	for d := range msgs {
		jobs <- d
	}

	wg.Wait()
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan amqp.Delivery, pgConnection *database.PostgresConnection, minioConnection *database.MinioConnection, qdrantConnection *database.QdrantConnection) {
	defer wg.Done()
	for d := range jobs {
		var embedMsg models.ReadmeEmbedMessage
		if err := json.Unmarshal(d.Body, &embedMsg); err != nil {
			log.Printf("Worker %d: Failed to unmarshal message: %v", id, err)
			d.Ack(false)
			continue
		}

		log.Printf("Worker %d: Received message to embed README for repository ID: %d", id, embedMsg.RepositoryID)

		// 1. Query PostgreSQL for repository metadata (pushed_at)
		repo, err := pgConnection.GetRepositoryByID(embedMsg.RepositoryID)
		if err != nil {
			log.Printf("Worker %d: Failed to get repository by ID %d from Postgres: %v", id, embedMsg.RepositoryID, err)
			d.Nack(false, true) // Requeue for retry
			continue
		}

		// 3. Fetch README content from MinIO
		readmeContent, err := minioConnection.GetFile(context.Background(), embedMsg.MinioPath)
		if err != nil {
			log.Printf("Worker %d: Failed to get README from MinIO for %s: %v", id, embedMsg.MinioPath, err)
			// 4. Fallback: Download README directly if not in MinIO
			if repo.ReadmeURL.Valid && repo.ReadmeURL.String != "" {
				log.Printf("Worker %d: Attempting to download README from %s", id, repo.ReadmeURL.String)
				resp, httpErr := http.Get(repo.ReadmeURL.String)
				if httpErr != nil || resp.StatusCode != http.StatusOK {
					log.Printf("Worker %d: Failed to download README from URL %s: %v", id, repo.ReadmeURL.String, httpErr)
					d.Ack(false)
					continue
				}
				defer resp.Body.Close()
				readmeContent, err = io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Worker %d: Failed to read downloaded README content: %v", id, err)
					d.Ack(false)
					continue
				}
			} else {
				log.Printf("Worker %d: No README URL available for repo %d, skipping embedding.", id, embedMsg.RepositoryID)
				d.Ack(false)
				continue
			}
		}

		// 5. Generate embedding (Placeholder - actual model integration needed)
		// For now, create a dummy embedding
		embedding := generateEmbedding(readmeContent)

		// 6. Save embedding to Qdrant
		points := []*qdrant_go_client.PointStruct{
			{
				Id:      &qdrant_go_client.PointId{PointIdOptions: &qdrant_go_client.PointId_Num{Num: uint64(embedMsg.RepositoryID)}},
				Vectors: &qdrant_go_client.Vectors{VectorsOptions: &qdrant_go_client.Vectors_Vector{Vector: &qdrant_go_client.Vector{Data: embedding}}},
			},
		}
		if err := qdrantConnection.UpsertVectors(context.Background(), "repositories", points); err != nil {
			log.Printf("Worker %d: Failed to insert embedding for repo %d into Qdrant: %v", id, embedMsg.RepositoryID, err)
			d.Nack(false, true) // Requeue for retry
			continue
		}

		log.Printf("Worker %d: Successfully processed and embedded README for repository ID: %d", id, embedMsg.RepositoryID)
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
