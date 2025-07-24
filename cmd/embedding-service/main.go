package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	qdrant_go_client "github.com/qdrant/go-client/qdrant"
	"github.com/streadway/amqp"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	numWorkers = 20
	queueName  = "readme_to_embed"
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
	err = qdrantConnection.CreateCollection(context.Background(), "repositories", vectorSize)
	if err != nil {
		log.Fatalf("Failed to create or verify Qdrant collection: %v", err)
	}

	jobs := make(chan amqp.Delivery, numWorkers)
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg, jobs, pgConnection, minioConnection, qdrantConnection)
	}

	log.Printf("Embedding service started. Waiting for messages on queue: %s", queueName)

	// Main loop to consume messages and handle reconnects
	for {
		msgs, err := mqConnection.Consume(queueName)
		if err != nil {
			log.Printf("Failed to start consuming from queue %s: %v. Reconnecting...", queueName, err)
			// Wait before trying to reconnect
			time.Sleep(5 * time.Second)
			mqConnection, err = messaging.NewConnection(cfg.RabbitMQURL)
			if err != nil {
				log.Printf("Failed to reconnect to RabbitMQ: %v", err)
				continue // try again
			}
			continue
		}

		// This channel will be closed when the connection is lost
		notifyClose := make(chan *amqp.Error)
		mqConnection.Connection.NotifyClose(notifyClose)

		log.Println("Successfully connected to RabbitMQ. Consuming messages.")

		// Process messages until the connection is lost
		for {
			select {
			case d, ok := <-msgs:
				if !ok {
					log.Println("Message channel closed. Reconnecting...")
					goto Reconnect
				}
				jobs <- d
			case err := <-notifyClose:
				log.Printf("Connection to RabbitMQ lost: %v. Reconnecting...", err)
				goto Reconnect
			}
		}

	Reconnect:
		log.Println("Attempting to reconnect to RabbitMQ...")
		// Close the old connection just in case
		mqConnection.Close()
		// Wait a bit before trying to establish a new connection
		time.Sleep(5 * time.Second)
		// Attempt to create a new connection
		newMqConnection, err := messaging.NewConnection(cfg.RabbitMQURL)
		if err != nil {
			log.Printf("Failed to reconnect to RabbitMQ: %v", err)
			// Continue the outer loop to try again
			continue
		}
		mqConnection = newMqConnection
	}
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan amqp.Delivery, pgConnection *database.PostgresConnection, minioConnection *database.MinioConnection, qdrantConnection *database.QdrantConnection) {
	defer wg.Done()
	for j := range jobs {
		var embedMsg models.ReadmeEmbedMessage
		if err := json.Unmarshal(j.Body, &embedMsg); err != nil {
			log.Printf("Worker %d: Failed to unmarshal message: %v", id, err)
			j.Nack(false, false) // Nack and don't requeue
			continue
		}

		log.Printf("Worker %d: Received message for repository ID: %d", id, embedMsg.RepositoryID)
		processMessage(id, embedMsg, pgConnection, minioConnection, qdrantConnection)
		j.Ack(false) // Ack the message
		log.Printf("Worker %d: Finished processing message for repository ID: %d", id, embedMsg.RepositoryID)
	}
}

func processMessage(
	id int,
	embedMsg models.ReadmeEmbedMessage,
	pgConnection *database.PostgresConnection,
	minioConnection *database.MinioConnection,
	qdrantConnection *database.QdrantConnection,
) {
	// Explicitly nil the slice after use to help the GC
	var readmeContent []byte
	defer func() {
		readmeContent = nil
		runtime.GC()
	}()

	readmeContent, err := minioConnection.GetFile(context.Background(), embedMsg.MinioPath)
	if err != nil {
		log.Printf("Worker %d: Failed to get README from MinIO for %s: %v", id, embedMsg.MinioPath, err)
		if embedMsg.DownloadURL != "" {
			log.Printf("Worker %d: Attempting to download README from %s", id, embedMsg.DownloadURL)
			resp, httpErr := http.Get(embedMsg.DownloadURL)
			if httpErr != nil || resp.StatusCode != http.StatusOK {
				log.Printf("Worker %d: Failed to download README from URL %s: %v", id, embedMsg.DownloadURL, httpErr)
				return
			}
			defer resp.Body.Close()
			readmeContent, err = io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Worker %d: Failed to read downloaded README content: %v", id, err)
				return
			}
			reader := bytes.NewReader(readmeContent)
			_, err := minioConnection.UploadFile(context.Background(), embedMsg.MinioPath, reader, int64(len(readmeContent)), "text/markdown")
			if err != nil {
				log.Printf("Worker %d: Failed to upload downloaded README to MinIO for %s: %v", id, embedMsg.MinioPath, err)
			} else {
				log.Printf("Worker %d: Successfully cached README for %s in MinIO", id, embedMsg.MinioPath)
			}
		} else {
			log.Printf("Worker %d: No README URL available for repo %d, skipping embedding.", id, embedMsg.RepositoryID)
			return
		}
	}

	embedding := generateEmbedding(readmeContent)

	points := []*qdrant_go_client.PointStruct{
		{
			Id:      &qdrant_go_client.PointId{PointIdOptions: &qdrant_go_client.PointId_Num{Num: uint64(embedMsg.RepositoryID)}},
			Vectors: &qdrant_go_client.Vectors{VectorsOptions: &qdrant_go_client.Vectors_Vector{Vector: &qdrant_go_client.Vector{Data: embedding}}},
		},
	}
	if err := qdrantConnection.UpsertVectors(context.Background(), "repositories", points); err != nil {
		log.Printf("Worker %d: Failed to insert embedding for repo %d into Qdrant: %v", id, embedMsg.RepositoryID, err)
		return
	}

	log.Printf("Worker %d: Successfully processed and embedded README for repository ID: %d", id, embedMsg.RepositoryID)
}

func generateEmbedding(content []byte) []float32 {
	requestBody, err := json.Marshal(map[string]string{
		"text": string(content),
	})
	if err != nil {
		log.Printf("Failed to marshal request body: %v", err)
		return nil
	}

	resp, err := http.Post("http://embedding-api-service/embed", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Failed to call embedding API: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Embedding API returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
		return nil
	}

	var result map[string][]float32
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode embedding response: %v", err)
		return nil
	}

	return result["embedding"]
}
