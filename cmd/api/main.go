package main

import (
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/api"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
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

	var redisClient *redis.Client
	for i := 0; i < maxRetries; i++ {
		redisClient, err = database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Redis: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to Redis after %d retries: %v", maxRetries, err)
	}

	var postgresConnection *database.PostgresConnection
	for i := 0; i < maxRetries; i++ {
		postgresConnection, err = database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to Postgres: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to Postgres after %d retries: %v", maxRetries, err)
	}
	postgresConnection.WithRedis(redisClient)

	var clickhouseConnection *database.ClickHouseConnection
	for i := 0; i < maxRetries; i++ {
		clickhouseConnection, err = database.NewClickHouseConnection(cfg.ClickHouseHost, cfg.ClickHousePort, cfg.ClickHouseUser, cfg.ClickHousePassword, cfg.ClickHouseDB)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to ClickHouse: %v. Retrying in %v...", err, retryDelay)
		time.Sleep(retryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse after %d retries: %v", maxRetries, err)
	}

	server := api.NewServer(cfg, redisClient, postgresConnection, clickhouseConnection)

	log.Println("API Server started. Listening on :8080")
	log.Fatal(server.Run(":8080"))
}
