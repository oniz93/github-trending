package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/github"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	maxStars   = 1000000 // A reasonable upper bound for stars
	minStars   = 50      // The minimum stars to be considered
	maxRetries = 5
	retryDelay = 5 * time.Second
)

var (
	githubClient   *github.GitHubClient
	mqConnection   messaging.MQConnection
	crawlQueueName = "repos_to_crawl"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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

	githubClient = github.NewGitHubClient(cfg.GitHubToken, nil)

	log.Println("Discovery service started.")

	// Run discovery on startup
	runDiscovery()

	// Then run discovery every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runDiscovery()
	}
}

func runDiscovery() {
	log.Println("Starting discovery process...")
	discoverRepositories(minStars, maxStars)
	log.Println("Finished a discovery cycle. Waiting for the next one.")
}

func discoverRepositories(minQuery, maxQuery int) {
	if minQuery > maxQuery {
		return
	}

	// Base case: If the range has collapsed, fetch results for the single star count.
	if minQuery == maxQuery {
		query := fmt.Sprintf("stars:%d", minQuery)
		log.Printf("Fetching repositories with exactly %d stars...", minQuery)
		fetchAllAndPublish(query)
		return
	}

	query := fmt.Sprintf("stars:%d..%d", minQuery, maxQuery)
	log.Printf("Searching for repositories with query: %s", query)

	searchResult, err := githubClient.SearchRepositories(query, 1)
	if err != nil {
		log.Printf("Failed to search repositories with query '%s': %v", query, err)
		return
	}

	if searchResult.TotalCount > 1000 {
		mid := minQuery + (maxQuery-minQuery)/2
		discoverRepositories(minQuery, mid)
		discoverRepositories(mid+1, maxQuery)
	} else {
		fetchAllAndPublish(query)
	}
}

func fetchAllAndPublish(query string) {
	page := 1
	for {
		searchResult, err := githubClient.SearchRepositories(query, page)
		if err != nil {
			log.Printf("Failed to fetch page %d for query '%s': %v", page, query, err)
			break
		}

		if len(searchResult.Items) == 0 {
			break // No more items
		}

		for _, repo := range searchResult.Items {
			// Create a DiscoveryMessage with the repository and the current time
			discoveryMessage := models.DiscoveryMessage{
				Repository:   repo,
				DiscoveredAt: time.Now(),
			}

			repoJSON, err := json.Marshal(discoveryMessage)
			if err != nil {
				log.Printf("Failed to marshal repository %s: %v", repo.FullName, err)
				continue
			}

			err = mqConnection.Publish(crawlQueueName, repoJSON)
			if err != nil {
				log.Printf("Failed to publish message for %s: %v", repo.FullName, err)
			} else {
				log.Printf("Published message to crawl: %s", repo.FullName)
			}
		}

		// If we have fewer results than the max per page, we are on the last page
		if len(searchResult.Items) < 100 {
			break
		}

		page++
	}
}
