package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/github"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	maxStars        = 1000000 // A reasonable upper bound for stars
	minStars        = 50      // The minimum stars to be considered
	lockFilePath    = "/tmp/discovery.lock"
	triggerFilePath = "/root/trigger_discovery"
)

var (
	githubClient   *github.GitHubClient
	mqConnection   *messaging.Connection
	crawlQueueName = "repos_to_crawl"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	mqConnection, err = messaging.NewConnection(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer mqConnection.Close()

	githubClient = github.NewGitHubClient(cfg.GitHubToken)

	log.Println("Discovery service started.")

	for {
		if fileExists(triggerFilePath) {
			log.Println("Trigger file found. Starting discovery immediately.")
			if err := os.Remove(triggerFilePath); err != nil {
				log.Printf("Failed to remove trigger file: %v", err)
			}
			discoverRepositories(minStars, maxStars)
		} else {
			minLock, maxLock, err := readLockFile()
			if err != nil {
				log.Printf("No lock file found. Waiting until midnight to start.")
				waitUntilMidnight()
				minLock, maxLock = minStars, maxStars
			} else {
				log.Printf("Found lock file. Resuming discovery from %d to %d stars.", minLock, maxLock)
			}
			discoverRepositories(minLock, maxLock)
		}

		if err := removeLockFile(); err != nil {
			log.Printf("Failed to remove lock file: %v", err)
		}

		log.Println("Finished a discovery cycle. Waiting for the next one.")
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func discoverRepositories(minQuery, maxQuery int) {
	if minQuery > maxQuery {
		return
	}

	if err := writeLockFile(minQuery, maxQuery); err != nil {
		log.Printf("Failed to write lock file: %v", err)
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

func writeLockFile(minQuery, maxQuery int) error {
	return ioutil.WriteFile(lockFilePath, []byte(fmt.Sprintf("%d,%d", minQuery, maxQuery)), 0644)
}

func readLockFile() (int, int, error) {
	data, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Split(string(data), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid lock file format")
	}

	minQuery, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	maxQuery, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return minQuery, maxQuery, nil
}

func removeLockFile() error {
	return os.Remove(lockFilePath)
}

func waitUntilMidnight() {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	time.Sleep(midnight.Sub(now))
}
