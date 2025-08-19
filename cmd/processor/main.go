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
	defer chConnection.DB.Close()

	var minioConnection *database.MinioConnection
	for i := 0; i < maxRetries; i++ {
		minioConnection, err = database.NewMinioConnection(cfg.MinioEndpoint, cfg.MinioRootUser, cfg.MinioRootPassword)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to MinIO: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to MinIO after %d retries: %v", maxRetries, err)
	}

	githubClient := github.NewGitHubClient(cfg.GitHubToken, http.DefaultClient)

	processQueueName := "raw_data_to_process"
	readmeEmbedQueueName := "readme_to_embed"

	err = processor.RunProcessor(mqConnection, pgConnection, chConnection, minioConnection, githubClient, http.DefaultClient, processQueueName, readmeEmbedQueueName)
	if err != nil {
		log.Fatalf("Processor service failed: %v", err)
	}
}
