package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	maxRetries               = 5
	retryDelay               = 5 * time.Second
	writeQueueName           = "repos_to_write"
	writeDeadLetterQueueName = "repos_to_write_dlq"
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
	pgConnection.WithRedis(redisClient)

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

	msgs, err := mqConnection.Consume(writeQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming from queue %s: %v", writeQueueName, err)
	}

	log.Printf("Writer service started. Waiting for messages on queue: %s", writeQueueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			handleMessage(d, mqConnection, pgConnection, chConnection)
		}
		log.Println("Message channel closed. Exiting writer service.")
		close(forever)
	}()

	<-forever
}

func handleMessage(d amqp.Delivery, mqConnection messaging.MQConnection, pgConnection *database.PostgresConnection, chConnection *database.ClickHouseConnection) {
	var crawlResult models.CrawlResult
	if err := json.Unmarshal(d.Body, &crawlResult); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		d.Ack(false) // Acknowledge and discard malformed message
		return
	}

	log.Printf("Processing data for repository: %s", crawlResult.Repository.FullName)

	if err := pgConnection.InsertRepository(crawlResult.Repository, crawlResult.CrawledAt); err != nil {
		log.Printf("Failed to insert repository into PostgreSQL: %v", err)

		// Unique/foreign-key conflicts are permanent: retrying the same
		// payload can never succeed and would block the queue forever.
		if database.IsConstraintViolation(err) {
			if pubErr := mqConnection.Publish(writeDeadLetterQueueName, d.Body); pubErr != nil {
				log.Printf("Failed to move message to dead-letter queue %s: %v", writeDeadLetterQueueName, pubErr)
				d.Nack(false, true)
				return
			}

			log.Printf("Moved message for %s to dead-letter queue %s after a constraint violation", crawlResult.Repository.FullName, writeDeadLetterQueueName)
			d.Ack(false)
			return
		}

		d.Nack(false, true) // Nack and requeue for another attempt
		return
	}

	if err := chConnection.InsertRepositoryStats(crawlResult.Repository); err != nil {
		log.Printf("Failed to insert repository stats into ClickHouse: %v", err)
		d.Nack(false, true) // Nack and requeue for another attempt
		return
	}

	log.Printf("Successfully wrote data for: %s", crawlResult.Repository.FullName)
	d.Ack(false)
}
