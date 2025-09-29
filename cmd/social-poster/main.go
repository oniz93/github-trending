package main

import (
	"log"
	"time"

	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/social"
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
	defer chConnection.Close()

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

	twitterClient := social.NewTwitterClient(cfg.TwitterApiKey, cfg.TwitterApiSecretKey, cfg.TwitterAccessToken, cfg.TwitterAccessSecret)

	log.Println("Social Poster service started.")

	// Run on startup
	runPoster(pgConnection, chConnection, twitterClient)

	// Run every 4 hours
	ticker := time.NewTicker(4 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		runPoster(pgConnection, chConnection, twitterClient)
	}
}

func runPoster(pgConnection *database.PostgresConnection, chConnection *database.ClickHouseConnection, poster social.Poster) {
	log.Println("Starting social posting process...")

	// Get trending repositories from the last day
	trendingRepoIDs, err := chConnection.GetTrendingRepositoryIDsByGrowth(7)
	if err != nil {
		log.Printf("Failed to get trending repositories: %v", err)
		return
	}

	if len(trendingRepoIDs) == 0 {
		log.Println("No trending repositories found to post.")
		return
	}

	// Get data for the top trending repository
	repo, err := pgConnection.GetRepositoryByID(trendingRepoIDs[0])
	if err != nil {
		log.Printf("Failed to get repository data for ID %d: %v", trendingRepoIDs[0], err)
		return
	}

	// Post it
	err = poster.Post(repo)
	if err != nil {
		log.Printf("Failed to post to social media: %v", err)
	}

	log.Println("Finished social posting cycle.")
}
