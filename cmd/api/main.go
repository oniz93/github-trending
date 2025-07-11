package main

import (
	"log"

	"github.com/teomiscia/github-trending/internal/api"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	postgresConnection, err := database.NewPostgresConnection(cfg.PostgresHost, "5432", cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	clickhouseConnection, err := database.NewClickHouseConnection(cfg.ClickhouseHost, cfg.ClickhousePort, cfg.ClickhouseUser, cfg.ClickhousePassword, cfg.ClickhouseDB)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	server := api.NewServer(redisClient, postgresConnection, clickhouseConnection)

	log.Println("API Server started. Listening on :8080")
	log.Fatal(server.Run(":8080"))
}
