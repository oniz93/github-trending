package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

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

	clickhousePort, _ := strconv.Atoi(cfg.ClickHousePort)
	chConnection, err := database.NewClickHouseConnection(cfg.ClickHouseHost, clickhousePort, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.ClickHouseDB)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chConnection.DB.Close()

	minioConnection, err := database.NewMinioConnection(cfg.MinioEndpoint, cfg.MinioRootUser, cfg.MinioRootPassword)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	processQueueName := "raw_data_to_process"
	readmeEmbedQueueName := "readme_to_embed"

	msgs, err := mqConnection.Consume(processQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming from queue %s: %v", processQueueName, err)
	}

	log.Printf("Processor service started. Waiting for raw data to process on queue: %s", processQueueName)

	for d := range msgs {
		var crawlResult models.CrawlResult
		if err := json.Unmarshal(d.Body, &crawlResult); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			d.Ack(false)
			continue
		}

		log.Printf("Processing data for repository: %s", crawlResult.Repository.FullName)

		if err := pgConnection.InsertRepository(crawlResult.Repository, crawlResult.CrawledAt); err != nil {
			log.Printf("Failed to insert repository into PostgreSQL: %v", err)
			d.Ack(false)
			continue
		}

		if err := chConnection.InsertRepositoryStats(crawlResult.Repository, crawlResult.DiscoveredAt); err != nil {
			log.Printf("Failed to insert repository stats into ClickHouse: %v", err)
			d.Ack(false)
			continue
		}

		// Handle README storage and embedding trigger
		if crawlResult.Repository.ReadmeURL != "" {
			readmeContent, err := downloadReadme(crawlResult.Repository.ReadmeURL)
			if err != nil {
				log.Printf("Failed to download README for %s: %v", crawlResult.Repository.FullName, err)
				// Continue processing, but don't trigger embedding for this README
			} else {
				objectName := fmt.Sprintf("readmes/%d.md", crawlResult.Repository.ID)
				reader := bytes.NewReader(readmeContent)
				_, err := minioConnection.UploadFile(context.Background(), objectName, reader, int64(len(readmeContent)), "text/markdown")
				if err != nil {
					log.Printf("Failed to upload README to MinIO for %s: %v", crawlResult.Repository.FullName, err)
					// Continue processing, but don't trigger embedding for this README
				} else {
					// Publish message to readme_to_embed queue
					embedMsg := models.ReadmeEmbedMessage{
						RepositoryID: int64(crawlResult.Repository.ID),
						MinioPath:    objectName,
					}
					embedMsgJSON, err := json.Marshal(embedMsg)
					if err != nil {
						log.Printf("Failed to marshal embed message for %s: %v", crawlResult.Repository.FullName, err)
					} else {
						err = mqConnection.Publish(readmeEmbedQueueName, embedMsgJSON)
						if err != nil {
							log.Printf("Failed to publish embed message for %s: %v", crawlResult.Repository.FullName, err)
						}
					}
				}
			}
		}

		log.Printf("Successfully processed and stored data for: %s", crawlResult.Repository.FullName)

		d.Ack(false)
	}
}

func downloadReadme(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download README: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download README, status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read README content: %w", err)
	}

	return content, nil
}
