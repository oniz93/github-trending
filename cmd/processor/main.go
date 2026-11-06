package main

import (
	"log"
	"net/http"
	"time"

	"github.com/teomiscia/github-trending/cmd/processor/processor"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/github"
	"github.com/teomiscia/github-trending/internal/messaging"
)

const (
	maxRetries = 5
	retryDelay = 5 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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

	var objectStore *database.S3Connection
	for i := 0; i < maxRetries; i++ {
		objectStore, err = database.NewS3Connection(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to S3 object store: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to S3 object store after %d retries: %v", maxRetries, err)
	}

	githubClient := github.NewGitHubClient(cfg.GitHubToken, http.DefaultClient)

	processQueueName := "raw_data_to_process"
	writeQueueName := "repos_to_write"
	readmeEmbedQueueName := "readme_to_embed"

	err = processor.RunProcessor(mqConnection, objectStore, githubClient, http.DefaultClient, processQueueName, writeQueueName, readmeEmbedQueueName)
	if err != nil {
		log.Fatalf("Processor service failed: %v", err)
	}
}
