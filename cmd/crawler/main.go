package main

import (
	"database/sql"
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

	var dbConnection *database.PostgresConnection
	for i := 0; i < maxRetries; i++ {
		dbConnection, err = database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to PostgreSQL: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL after %d retries: %v", maxRetries, err)
	}
	defer dbConnection.DB.Close()

	githubClient := github.NewGitHubClient(cfg.GitHubToken, nil)
	crawlQueueName := "repos_to_crawl"
	processQueueName := "raw_data_to_process"

	err = runCrawler(mqConnection, dbConnection, githubClient, http.DefaultClient, "https://raw.githubusercontent.com", crawlQueueName, processQueueName)
	if err != nil {
		log.Fatalf("Crawler service failed: %v", err)
	}
}

func runCrawler(mqConnection messaging.MQConnection, dbConnection database.DBConnection, githubClient *github.GitHubClient, httpClient *http.Client, rawContentBaseURL, crawlQueueName, processQueueName string) error {
	msgs, err := mqConnection.Consume(crawlQueueName)
	if err != nil {
		return fmt.Errorf("failed to start consuming from queue %s: %w", crawlQueueName, err)
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

		lastCrawledAt, err := dbConnection.GetLastCrawlTime(int64(repo.ID))
		if err != nil {
			log.Printf("Failed to get last crawl time for %s: %v", repo.FullName, err)
			// Decide if you want to continue or not
		}

		if repo.PushedAt.Before(lastCrawledAt) {
			log.Printf("Skipping %s, no new updates since last crawl.", repo.FullName)
			d.Ack(false)
			continue
		}

		readmeURL := findReadme(httpClient, &repo, rawContentBaseURL)
		if readmeURL != "" {
			repo.ReadmeURL = sql.NullString{String: readmeURL, Valid: true}
		} else {
			repo.ReadmeURL = sql.NullString{Valid: false}
		}
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
	return nil
}

func findReadme(client *http.IClient, repo *models.Repository, rawContentBaseURL string) string {
	readmeNames := []string{"README.md", "README.txt"}

	for _, name := range readmeNames {
		url := fmt.Sprintf("%s/%s/%s/%s", rawContentBaseURL, repo.FullName, repo.DefaultBranch, name)

		resp, err := client.Head(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("Found README for %s at %s", repo.FullName, url)
			resp.Body.Close()
			return url
		}

		if resp != nil {
			log.Printf("Checked for README at %s, status code: %d", url, resp.StatusCode)
			resp.Body.Close()
		} else {
			log.Printf("Error checking for README at %s: %v", url, err)
		}
	}

	log.Printf("README not found for %s", repo.FullName)
	return ""
}
