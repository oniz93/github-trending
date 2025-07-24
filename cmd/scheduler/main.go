package main

import (
	"log"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
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

	var mqConnection *messaging.RabbitMQConnection
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

	crawlQueueName := "repos_to_crawl"

	log.Println("Scheduler service started. Scheduling repository refreshes...")

	for {
		// In a real app, you'd query your database for repos to refresh.
		repoURL := "https://github.com/example/existing-repo"

		err := mqConnection.Publish(crawlQueueName, []byte(repoURL))
		if err != nil {
			log.Printf("Failed to publish message: %v", err)
		} else {
			log.Printf("Published message to crawl: %s", repoURL)
		}

		time.Sleep(30 * time.Second) // Run every 30 seconds for demonstration.
	}
}
