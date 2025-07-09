package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/teomiscia/github-trending/internal/github"
	"github.com/teomiscia/github-trending/internal/models"
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

// MockPostgresConnection simulates a PostgreSQL connection for testing.
type MockPostgresConnection struct {
	LastCrawlTime time.Time
}

func (m *MockPostgresConnection) GetLastCrawlTime(repoID int64) (time.Time, error) {
	return m.LastCrawlTime, nil
}

func (m *MockPostgresConnection) InsertRepository(repo models.Repository, lastCrawledAt time.Time) error {
	return nil // No-op for test
}

func (m *MockPostgresConnection) GetRepositoryByID(repoID int64) (models.Repository, error) {
	return models.Repository{}, nil // No-op for test
}

// Mock for the database.PostgresConnection.DB field (sql.DB)
func (m *MockPostgresConnection) Close() error {
	return nil
}

func TestCrawlerIntegration(t *testing.T) {
	// 1. Mock GitHub API Server
	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/eugeneware/gifencoder/tags":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `[{"name": "v1.0.0"}, {"name": "v0.9.0"}]`)
		case r.URL.Path == "/repos/eugeneware/gifencoder/languages":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"JavaScript": 10000, "HTML": 500}`)
		default:
			t.Errorf("Unexpected GitHub API request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockGitHubServer.Close()

	// 2. Mock raw.githubusercontent.com HTTP HEAD requests
	mockRawGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" && r.URL.Path == "/eugeneware/gifencoder/master/README.md" {
			w.WriteHeader(http.StatusOK)
		} else {
			t.Errorf("Unexpected raw.githubusercontent.com request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockRawGitHubServer.Close()

	// Override http.DefaultClient for findReadme
	// originalDefaultClient := http.DefaultClient
	// http.DefaultClient = mockRawGitHubServer.Client()
	// defer func() { http.DefaultClient = originalDefaultClient }()

	// 3. Mock RabbitMQ
	mockMQConnection := NewMockRabbitMQConnection()

	// 4. Mock PostgreSQL
	mockDBConnection := &MockPostgresConnection{
		LastCrawlTime: time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC), // Older than PushedAt
	}

	// 5. Initialize GitHub Client with mock server URL
	mockGitHubClient := github.NewGitHubClient("mock_token", mockGitHubServer.Client())
	mockGitHubClient.SetBaseURL(mockGitHubServer.URL)

	// 6. Prepare the input message
	sampleRepo := models.Repository{
		ID:           13329152,
		FullName:     "eugeneware/gifencoder",
		DefaultBranch: "master",
		PushedAt:     time.Date(2025, time.June, 6, 7, 9, 34, 0, time.UTC),
	}
	discoveryMsg := models.DiscoveryMessage{
		Repository:   sampleRepo,
		DiscoveredAt: time.Date(2025, time.July, 9, 2, 0, 15, 0, time.UTC),
	}
	msgBody, err := json.Marshal(discoveryMsg)
	if err != nil {
		t.Fatalf("Failed to marshal discovery message: %v", err)
	}

	// Simulate message consumption by pushing to the mock consume channel
	mockMQConnection.consumeChan <- amqp.Delivery{
		Body: msgBody,
		Acknowledger: &mockAcknowledger{}, // Provide a mock acknowledger
	}

	// Run the crawler in a goroutine
	go func() {
		err := runCrawler(mockMQConnection, mockDBConnection, mockGitHubClient, mockRawGitHubServer.Client(), mockRawGitHubServer.URL, "repos_to_crawl", "raw_data_to_process")
		if err != nil {
			t.Errorf("runCrawler returned an error: %v", err)
		}
	}()

	// Wait for the processed message to be published
	select {
	case publishedMsg := <-mockMQConnection.publishChan:
		var crawlResult models.CrawlResult
		if err := json.Unmarshal(publishedMsg.Body, &crawlResult); err != nil {
			t.Fatalf("Failed to unmarshal published message: %v", err)
		}

		// Assertions
		if crawlResult.Repository.ID != sampleRepo.ID {
			t.Errorf("Expected repository ID %d, got %d", sampleRepo.ID, crawlResult.Repository.ID)
		}
		if crawlResult.Repository.FullName != sampleRepo.FullName {
			t.Errorf("Expected repository FullName %s, got %s", sampleRepo.FullName, crawlResult.Repository.FullName)
		}
		if crawlResult.Repository.ReadmeURL.String != fmt.Sprintf("%s/eugeneware/gifencoder/master/README.md", mockRawGitHubServer.URL) {
			t.Errorf("Expected ReadmeURL %s, got %s", fmt.Sprintf("%s/eugeneware/gifencoder/master/README.md", mockRawGitHubServer.URL), crawlResult.Repository.ReadmeURL.String)
		}
		if len(crawlResult.Repository.Tags) != 2 || crawlResult.Repository.Tags[0] != "v1.0.0" {
			t.Errorf("Expected tags [v1.0.0, v0.9.0], got %v", crawlResult.Repository.Tags)
		}
		if crawlResult.Repository.Languages["JavaScript"] != 10000 || crawlResult.Repository.Languages["HTML"] != 500 {
			t.Errorf("Expected languages {\"JavaScript\": 10000, \"HTML\": 500}, got %v", crawlResult.Repository.Languages)
		}
		if crawlResult.DiscoveredAt != discoveryMsg.DiscoveredAt {
			t.Errorf("Expected DiscoveredAt %v, got %v", discoveryMsg.DiscoveredAt, crawlResult.DiscoveredAt)
		}
		if crawlResult.CrawledAt.IsZero() {
			t.Error("CrawledAt should not be zero")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for processed message")
	}
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