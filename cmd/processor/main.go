package main

import (
	"encoding/json"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
	"log"
	"strconv"
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

	processQueueName := "raw_data_to_process"

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

		log.Printf("Successfully processed and stored data for: %s", crawlResult.Repository.FullName)

		d.Ack(false)
	}
}
