package processor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/github"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

func RunProcessor(mqConnection messaging.MQConnection, pgConnection database.DBConnection, chConnection database.CHConnection, minioConnection database.MinioClient, githubClient *github.GitHubClient, httpClient *http.Client, processQueueName, readmeEmbedQueueName string) error {
	msgs, err := mqConnection.Consume(processQueueName)
	if err != nil {
		return fmt.Errorf("failed to start consuming from queue %s: %w", processQueueName, err)
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

		// Get README URL
		readmeURL, err := githubClient.GetReadme(crawlResult.Repository.FullName)
		if err != nil {
			log.Printf("Failed to get README for %s: %v", crawlResult.Repository.FullName, err)
		} else {
			crawlResult.Repository.ReadmeURL = sql.NullString{String: readmeURL, Valid: true}
		}

		if err := pgConnection.InsertRepository(crawlResult.Repository, crawlResult.CrawledAt); err != nil {
			log.Printf("Failed to insert repository into PostgreSQL: %v", err)
			d.Ack(false)
			continue
		}

		if err := chConnection.InsertRepositoryStats(crawlResult.Repository); err != nil {
			log.Printf("Failed to insert repository stats into ClickHouse: %v", err)
			d.Ack(false)
			continue
		}

		// Handle README storage and embedding trigger
		if crawlResult.Repository.ReadmeURL.Valid {
			readmeContent, err := downloadReadme(httpClient, crawlResult.Repository.ReadmeURL.String)
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
						DownloadURL:  crawlResult.Repository.ReadmeURL.String,
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
	return nil
}

func downloadReadme(httpClient *http.Client, url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
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
