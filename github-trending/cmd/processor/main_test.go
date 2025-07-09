package main_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/streadway/amqp"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/messaging"
	"github.com/teomiscia/github-trending/internal/models"
	processor "github.com/teomiscia/github-trending/cmd/processor/processor"
)

// MockRabbitMQConnection simulates a RabbitMQ connection for testing.
type MockRabbitMQConnection struct {
	consumeChan chan amqp.Delivery
	publishChan chan amqp.Publishing
	mu          sync.Mutex
}

func NewMockRabbitMQConnection() *MockRabbitMQConnection {
	return &MockRabbitMQConnection{
		consumeChan: make(chan amqp.Delivery, 100), // Buffered channel
		publishChan: make(chan amqp.Publishing, 100),
	}
}

func (m *MockRabbitMQConnection) Consume(queueName string) (<-chan amqp.Delivery, error) {
	// In a real scenario, you might want to map queueName to different channels.
	// For this test, we'll just use one consume channel.
	return m.consumeChan, nil
}

func (m *MockRabbitMQConnection) Publish(queueName string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishChan <- amqp.Publishing{
		Body: body,
	}
	return nil
}

func (m *MockRabbitMQConnection) Close() error {
	close(m.consumeChan)
	close(m.publishChan)
	return nil
}

// mockAcknowledger implements amqp.Acknowledger for testing purposes.
type mockAcknowledger struct{}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

func TestProcessorIntegration(t *testing.T) {
	// Setup dockertest pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// --- PostgreSQL Container Setup ---
	var pgResource *dockertest.Resource
	pgResource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "17.5-alpine",
		Env: []string{
			"POSTGRES_USER=user",
			"POSTGRES_PASSWORD=password",
			"POSTGRES_DB=github_meta",
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {{HostIP: "", HostPort: "5432"}},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start PostgreSQL container: %s", err)
	}
	defer func() {
		if err := pool.Purge(pgResource); err != nil {
			log.Fatalf("Could not purge PostgreSQL resource: %s", err)
		}
	}()

	pgHostPort := pgResource.GetPort("5432/tcp")
	pgConnStr := fmt.Sprintf("host=%s port=%s user=user password=password dbname=github_meta sslmode=disable", "localhost", pgHostPort)

	// Wait for PostgreSQL to be ready
	var pgDB *sql.DB
	err = pool.Retry(func() error {
		pgDB, err = sql.Open("postgres", pgConnStr)
		if err != nil {
			return err
		}
		return pgDB.Ping()
	})
	if err != nil {
		log.Fatalf("Could not connect to PostgreSQL: %s", err)
	}
	defer pgDB.Close()

	// Run PostgreSQL migrations
	pgSchema, err := os.ReadFile("../../storage/postgres/schema.sql")
	if err != nil {
		t.Fatalf("Failed to read PostgreSQL schema file: %v", err)
	}
	if _, err := pgDB.Exec(string(pgSchema)); err != nil {
		t.Fatalf("Failed to execute PostgreSQL schema: %v", err)
	}

	// --- ClickHouse Container Setup ---
	var chResource *dockertest.Resource
	chResource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "clickhouse/clickhouse-server",
		Tag:        "latest",
		Env: []string{
			"CLICKHOUSE_USER=clickhouse",
			"CLICKHOUSE_PASSWORD=clickhouse",
			"CLICKHOUSE_DB=default",
		},
		ExposedPorts: []string{"8123", "9000"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8123/tcp": {{HostIP: "", HostPort: "8123"}},
			"9000/tcp": {{HostIP: "", HostPort: "9000"}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start ClickHouse container: %s", err)
	}
	defer func() {
		if err := pool.Purge(chResource); err != nil {
			log.Fatalf("Could not purge ClickHouse resource: %s", err)
		}
	}()

	chHostPortTCP := chResource.GetPort("9000/tcp")

	// Wait for ClickHouse to be ready
	var chDB *sql.DB
	err = pool.Retry(func() error {
		chDB, err = sql.Open("clickhouse", fmt.Sprintf("tcp://localhost:%s?username=clickhouse&password=clickhouse&database=default", chHostPortTCP))
		if err != nil {
			return err
		}
		return chDB.Ping()
	})
	if err != nil {
		log.Fatalf("Could not connect to ClickHouse: %s", err)
	}
	defer chDB.Close()

	// Run ClickHouse migrations
	chSchema, err := os.ReadFile("../../storage/clickhouse/schema.sql")
	if err != nil {
		t.Fatalf("Failed to read ClickHouse schema file: %v", err)
	}
	if _, err := chDB.Exec(string(chSchema)); err != nil {
		t.Fatalf("Failed to execute ClickHouse schema: %v", err)
	}

	// --- MinIO Container Setup ---
	var minioResource *dockertest.Resource
	minioResource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/minio",
		Tag:        "latest",
		Env: []string{
			"MINIO_ROOT_USER=minioadmin",
			"MINIO_ROOT_PASSWORD=minioadmin",
		},
		Cmd:          []string{"server", "/data"},
		ExposedPorts: []string{"9000", "9001"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9000/tcp": {{HostIP: "", HostPort: "9000"}},
			"9001/tcp": {{HostIP: "", HostPort: "9001"}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start MinIO container: %s", err)
	}
	defer func() {
		if err := pool.Purge(minioResource); err != nil {
			log.Fatalf("Could not purge MinIO resource: %s", err)
		}
	}()

	minioHostPortAPI := minioResource.GetPort("9000/tcp")
	minioEndpoint := fmt.Sprintf("localhost:%s", minioHostPortAPI)

	// Wait for MinIO to be ready
	var minioClient *database.MinioConnection
	err = pool.Retry(func() error {
		minioClient, err = database.NewMinioConnection(minioEndpoint, "minioadmin", "minioadmin")
		if err != nil {
			return err
		}
		// Try to create a bucket to ensure MinIO is fully up
		return minioClient.Client.MakeBucket(context.Background(), minioClient.Bucket, minio.MakeBucketOptions{})
	})
	if err != nil {
		log.Fatalf("Could not connect to MinIO: %s", err)
	}
	defer minioClient.Close()

	// --- RabbitMQ Container Setup ---
	var rabbitResource *dockertest.Resource
	rabbitResource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "management-alpine",
		Env: []string{
			"RABBITMQ_DEFAULT_USER=user",
			"RABBITMQ_DEFAULT_PASS=password",
		},
		ExposedPorts: []string{"5672"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5672/tcp": {{HostIP: "", HostPort: "5672"}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start RabbitMQ container: %s", err)
	}
	defer func() {
		if err := pool.Purge(rabbitResource); err != nil {
			log.Fatalf("Could not purge RabbitMQ resource: %s", err)
		}
	}()

	rabbitHostPort := rabbitResource.GetPort("5672/tcp")
	rabbitMQURL := fmt.Sprintf("amqp://user:password@localhost:%s/", rabbitHostPort)

	// Wait for RabbitMQ to be ready
	var mqConnection *messaging.Connection
	err = pool.Retry(func() error {
		mqConnection, err = messaging.NewConnection(rabbitMQURL)
		if err != nil {
			return err
		}
		return mqConnection.Close() // Just check if we can connect and close
	})
	if err != nil {
		log.Fatalf("Could not connect to RabbitMQ: %s", err)
	}
	defer mqConnection.Close()

	// --- Mock HTTP Server for README Download ---
	mockReadmeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "README.md") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "# Test README Content")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockReadmeServer.Close()

	// --- Initialize Processor Dependencies ---
	pgConn, err := database.NewPostgresConnection("localhost", pgHostPort, "user", "password", "github_meta")
	if err != nil {
		t.Fatalf("Failed to create Postgres connection: %v", err)
	}
	defer pgConn.Close()

	chConn, err := database.NewClickHouseConnection("localhost", chHostPortTCP, "clickhouse", "clickhouse", "default")
	if err != nil {
		t.Fatalf("Failed to create ClickHouse connection: %v", err)
	}
	defer chConn.Close()

	// --- Prepare Input Message ---
	sampleCrawlResult := models.CrawlResult{
		Repository: models.Repository{
			ID:           76329726,
			NodeID:       sql.NullString{String: "MDEwOlJlcG9zaXRvcnk3NjMyOTcyNg==", Valid: true},
			Name:         "scheme-lib",
			FullName:     "evilbinary/scheme-lib",
			Owner: models.Owner{
				Login:     "evilbinary",
				ID:        5143386,
				NodeID:    sql.NullString{String: "MDQ6VXNlcjUxNDMzODY=", Valid: true},
				AvatarURL: "https://avatars.githubusercontent.com/u/5143386?v=4",
				HTMLURL:   "https://github.com/evilbinary",
				Type:      "User",
				SiteAdmin: sql.NullBool{Bool: false, Valid: true},
			},
			Private:       false,
			HTMLURL:       "https://github.com/evilbinary/scheme-lib",
			Description:   sql.NullString{String: "鸭库 duck lib scheme for gui gles gl slib openal socket web mongodb box2d game glfw mysql libevent  libuv uv json http client  server android osx linux chezscheme scheme-lib ", Valid: true},
			Fork:          false,
			URL:           "https://api.github.com/repos/evilbinary/scheme-lib",
			CreatedAt:     time.Date(2016, 12, 13, 6, 27, 36, 0, time.UTC),
			UpdatedAt:     time.Date(2025, 6, 26, 6, 53, 17, 0, time.UTC),
			PushedAt:      time.Date(2024, 2, 27, 10, 51, 7, 0, time.UTC),
			Homepage:      sql.NullString{String: "https://evilbinary.github.io/scheme-lib/ ", Valid: true},
			Size:          81426,
			StargazersCount: 473,
			WatchersCount: 473,
			Language:      sql.NullString{String: "Scheme", Valid: true},
			ForksCount:    48,
			OpenIssuesCount: 0,
			DefaultBranch: "master",
			Score:         1,
			Topics:        []string{"android", "chezscheme", "cl", "gui", "library", "lisp", "scheme", "scheme-lib", "ui"},
			HasIssues:     true,
			HasProjects:   true,
			HasPages:      true,
			HasWiki:       true,
			HasDownloads:  true,
			HasDiscussions: true,
			Archived:      false,
			Disabled:      false,
			Visibility:    "public",
			License: models.License{
				Key:    sql.NullString{String: "mit", Valid: true},
				Name:   sql.NullString{String: "MIT License", Valid: true},
				SpdxID: sql.NullString{String: "MIT", Valid: true},
				URL:    sql.NullString{String: "https://api.github.com/licenses/mit", Valid: true},
				NodeID: sql.NullString{String: "MDc6TGljZW5zZTEz", Valid: true},
			},
			AllowForking:  true,
			IsTemplate:    false,
			ReadmeURL:     sql.NullString{String: mockReadmeServer.URL + "/evilbinary/scheme-lib/master/README.md", Valid: true},
			Tags:          []string{"2.2", "2.1", "2.0", "1.0"},
			Languages: map[string]int{
				"Batchfile":  269,
				"C":          2342363,
				"C++":        1004842,
				"CSS":        38,
				"HTML":       1195,
				"JavaScript": 1647,
				"Lex":        922,
				"M4":         4007,
				"Makefile":   56419,
				"NSIS":       21781,
				"PostScript": 15140,
				"Python":     3621,
				"Roff":       1275,
				"Scheme":     6208896,
				"Shell":      10266,
			},
			LastCrawledAt: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		DiscoveredAt: time.Date(2025, 7, 9, 2, 0, 15, 491083674, time.UTC),
		CrawledAt:    time.Date(2025, 7, 9, 5, 39, 48, 780484383, time.UTC),
	}
	msgBody, err := json.Marshal(sampleCrawlResult)
	if err != nil {
		t.Fatalf("Failed to marshal crawl result: %v", err)
	}

	// --- Run Processor ---
	processQueueName := "raw_data_to_process"
	readmeEmbedQueueName := "readme_to_embed"

	// Simulate message consumption by pushing to the mock consume channel
	mqConnection.Consume(processQueueName) // Ensure queue is declared
	mqConnection.Publish(processQueueName, msgBody)

	// Run the processor in a goroutine
	go func() {
		err := processor.RunProcessor(mqConnection, pgConn, chConn, minioClient, mockReadmeServer.Client(), processQueueName, readmeEmbedQueueName)
		if err != nil {
			t.Errorf("RunProcessor returned an error: %v", err)
		}
	}()

	// Wait for processing to complete (e.g., by consuming from the output queue or a timeout)
	select {
	case publishedMsgChan, ok := <-mqConnection.Consume(readmeEmbedQueueName):
		if !ok {
			t.Fatal("RabbitMQ consume channel closed unexpectedly")
		}
		var embedMsg models.ReadmeEmbedMessage
		if err := json.Unmarshal(publishedMsgChan.Body, &embedMsg); err != nil {
			t.Fatalf("Failed to unmarshal embed message: %v", err)
		}
		// Assertions for embed message
		if embedMsg.RepositoryID != sampleCrawlResult.Repository.ID {
			t.Errorf("Expected embed RepositoryID %d, got %d", sampleCrawlResult.Repository.ID, embedMsg.RepositoryID)
		}
		if embedMsg.MinioPath != fmt.Sprintf("readmes/%d.md", sampleCrawlResult.Repository.ID) {
			t.Errorf("Expected embed MinioPath %s, got %s", fmt.Sprintf("readmes/%d.md", sampleCrawlResult.Repository.ID), embedMsg.MinioPath)
		}

	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for processed message or embed message")
	}

	// --- Verification of Data in Databases ---
	// PostgreSQL Verification
	pgRepo, err := pgConn.GetRepositoryByID(int64(sampleCrawlResult.Repository.ID))
	if err != nil {
		t.Fatalf("Failed to get repository from Postgres: %v", err)
	}

	// Deep comparison of relevant fields
	if pgRepo.ID != sampleCrawlResult.Repository.ID ||
		pgRepo.FullName != sampleCrawlResult.Repository.FullName ||
		pgRepo.Description.String != sampleCrawlResult.Repository.Description.String ||
		pgRepo.StargazersCount != sampleCrawlResult.Repository.StargazersCount ||
		pgRepo.ForksCount != sampleCrawlResult.Repository.ForksCount ||
		pgRepo.OpenIssuesCount != sampleCrawlResult.Repository.OpenIssuesCount ||
		pgRepo.DefaultBranch != sampleCrawlResult.Repository.DefaultBranch ||
		pgRepo.HTMLURL != sampleCrawlResult.Repository.HTMLURL {
		t.Errorf("PostgreSQL repository data mismatch. Expected %+v, got %+v", sampleCrawlResult.Repository, pgRepo)
	}

	// Verify languages
	if len(pgRepo.Languages) != len(sampleCrawlResult.Repository.Languages) {
		t.Errorf("PostgreSQL languages count mismatch. Expected %d, got %d", len(sampleCrawlResult.Repository.Languages), len(pgRepo.Languages))
	} else {
		for lang, size := range sampleCrawlResult.Repository.Languages {
			if pgRepo.Languages[lang] != size {
				t.Errorf("PostgreSQL language %s size mismatch. Expected %d, got %d", lang, size, pgRepo.Languages[lang])
			}
		}
	}

	// Verify tags
	if len(pgRepo.Tags) != len(sampleCrawlResult.Repository.Tags) {
		t.Errorf("PostgreSQL tags count mismatch. Expected %d, got %d", len(sampleCrawlResult.Repository.Tags), len(pgRepo.Tags))
	} else {
		for _, tag := range sampleCrawlResult.Repository.Tags {
			found := false
			for _, pgTag := range pgRepo.Tags {
				if pgTag == tag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("PostgreSQL missing tag: %s", tag)
			}
		}
	}

	// ClickHouse Verification (simplified - just check if a record exists)
	// In a real scenario, you'd query for the specific stats and compare.
	var count int
	err = chDB.QueryRow("SELECT count() FROM repository_stats WHERE repository_id = ?", sampleCrawlResult.Repository.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query ClickHouse: %v", err)
	}
	if count == 0 {
		t.Errorf("No stats found for repository %d in ClickHouse", sampleCrawlResult.Repository.ID)
	}

	// MinIO Verification
	minioContent, err := minioClient.GetFile(context.Background(), fmt.Sprintf("readmes/%d.md", sampleCrawlResult.Repository.ID))
	if err != nil {
		t.Fatalf("Failed to get README from MinIO: %v", err)
	}
	if string(minioContent) != "# Test README Content" {
		t.Errorf("MinIO README content mismatch. Expected \"# Test README Content\", got \"%s\"", string(minioContent))
	}
}
