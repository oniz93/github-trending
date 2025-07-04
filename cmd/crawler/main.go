package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/github"
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

	dbConnection, err := database.NewPostgresConnection(cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer dbConnection.DB.Close()

	githubClient := github.NewGitHubClient(cfg.GitHubToken)
	crawlQueueName := "repos_to_crawl"
	processQueueName := "raw_data_to_process"

	msgs, err := mqConnection.Consume(crawlQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming from queue %s: %v", crawlQueueName, err)
	}

	log.Printf("Crawler service started. Waiting for messages on queue: %s", crawlQueueName)

	for d := range msgs {
		var msg models.DiscoveryMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			d.Ack(false)
			continue
		}

		repo := msg.Repository
		log.Printf("Received a message to crawl: %s", repo.FullName)

		lastCrawledAt, err := dbConnection.GetLastCrawlTime(repo.ID)
		if err != nil {
			log.Printf("Failed to get last crawl time for %s: %v", repo.FullName, err)
			// Decide if you want to continue or not
		}

		if repo.PushedAt.Before(lastCrawledAt) {
			log.Printf("Skipping %s, no new updates since last crawl.", repo.FullName)
			d.Ack(false)
			continue
		}

		repo.ReadmeURL = findReadme(&repo)
		repo.Tags, err = githubClient.GetTags(repo.FullName)
		if err != nil {
			log.Printf("Failed to get tags for %s: %v", repo.FullName, err)
			d.Nack(false, true) // Requeue the message
			continue
		}
		repo.Languages, err = githubClient.GetLanguages(repo.FullName)
		if err != nil {
			log.Printf("Failed to get languages for %s: %v", repo.FullName, err)
			d.Nack(false, true) // Requeue the message
			continue
		}

		crawlResult := models.CrawlResult{
			Repository:   repo,
			DiscoveredAt: msg.DiscoveredAt,
			CrawledAt:    time.Now(),
		}

		resultJSON, err := json.Marshal(crawlResult)
		if err != nil {
			log.Printf("Failed to marshal crawl result for %s: %v", repo.FullName, err)
			d.Ack(false)
			continue
		}

		err = mqConnection.Publish(processQueueName, resultJSON)
		if err != nil {
			log.Printf("Failed to publish raw data for %s: %v", repo.FullName, err)
			continue
		}

		log.Printf("Successfully crawled and published data for: %s", repo.FullName)
		d.Ack(false)
	}
}

func findReadme(repo *models.Repository) string {
	readmeNames := []string{"README.md", "README.txt"}

	for _, name := range readmeNames {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/refs/heads/%s/%s", repo.FullName, repo.DefaultBranch, name)

		resp, err := http.Head(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return url
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return ""
}
