Excellent. Let's get you building. This is the most exciting phase of any project.

This guide will provide a complete blueprint for your local development setup on macOS using Docker and Go. We will define the architecture, the project structure, and provide the initial configuration files and code stubs to get you started immediately.

### Part 1: The Core Architecture - A System of Microservices

Your intuition is spot on. We need to break this system down into small, independent services that each do one thing well. This makes the system easier to develop, test, debug, and scale. We'll add a couple more services to your initial list for a cleaner separation of concerns.

Here are the seven core microservices we will build:

1.  **`discovery-service`**: Its only job is to find *new* repositories to track. It will run periodically (e.g., once a day), use the GitHub Search API to find repositories matching our criteria (e.g., `stars:>50`), and push their URLs to a queue for the crawlers.
2.  **`scheduler-service`**: Its only job is to schedule *refreshes* for repositories we already track. It will run continuously, query our own database for repositories that haven't been updated recently, and push their URLs to the same queue as the discovery service.
3.  **`crawler-service`**: This is a simple worker. It pulls a repository URL from the queue, fetches all the necessary data from the GitHub API (stats, languages, README URL, etc.), and pushes the raw, unprocessed data to another queue.
4.  **`processor-service`**: This is another worker. It pulls raw data from the crawler's output queue, parses and cleans it, then saves it to the appropriate databases (time-series stats to ClickHouse, metadata to PostgreSQL, and READMEs to MinIO). It also triggers the `embedding-service` when a new README is processed.
5.  **`embedding-service`**: This service consumes messages about new READMEs, downloads their content (from MinIO or directly from GitHub), generates semantic embeddings using a pre-trained model, and stores these vectors in Milvus (our vector database).
6.  **`similarity-engine-service`**: This service runs periodically, fetches repository embeddings from Milvus, calculates similarity scores between repositories, and stores pre-computed similarity lists in Redis for fast retrieval by the API server.
7.  **`api-server`**: This is the public-facing service. It listens for HTTP requests from your web application, queries the databases to calculate trending data, and returns it as a clean JSON response. It now also provides personalized recommendations based on user interaction history, leveraging pre-computed similarity lists from Redis.

This design creates a robust, one-way data flow:

`Discovery/Scheduler` -> `Crawl Queue` -> `Crawler` -> `Process Queue` -> `Processor` -> `Databases` <- `API Server`
                                                                                                `Processor` -> `Embeddings Queue` -> `Embedding Service` -> `Milvus`
                                                                                                `Similarity Engine` -> `Milvus` -> `Redis`

### Part 2: Local Development Setup with Docker Compose

`docker-compose` is the perfect tool to orchestrate all these services and their dependencies locally. We'll create one file to define and link everything.

#### Step 1: Project Directory Structure

First, create a root directory for your project. Inside it, we'll have a `docker-compose.yml` file and a directory for each of our Go services.

```bash
mkdir github-trending
cd github-trending

# Create directories for each Go microservice
mkdir discovery-service
mkdir scheduler-service
mkdir crawler-service
mkdir processor-service
mkdir api-server
mkdir embedding-service
mkdir similarity-engine-service

# Create a directory for shared Go code
mkdir -p internal/database

# Create the docker-compose file
touch docker-compose.yml
```

#### Step 2: The `.env` File

Create a `.env` file in the root of your project to store environment variables common to all services:

```dotenv
RABBITMQ_URL=amqp://user:password@rabbitmq:5672/
GITHUB_TOKEN=your-github-token
GITHUB_CALLBACK_SECRET=your-github-callback-secret

# RabbitMQ Credentials
RABBITMQ_DEFAULT_USER=user
RABBITMQ_DEFAULT_PASS=password
RABBITMQ_ERLANG_COOKIE="ab3eLHd4xW1amBWjGqzDW6I9idDIFp9h4Tcbn80Iqg4="

# Postgres Credentials
POSTGRES_HOST=postgres
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=github_meta

# MinIO Credentials
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_ENDPOINT=minio:9000

# Clickhouse Credentials
CLICKHOUSE_HOST=clickhouse
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=clickhouse
CLICKHOUSE_PASSWORD=clickhouse
CLICKHOUSE_DB=default

# Redis Credentials
REDIS_HOST=redis_cache
REDIS_PORT=6379
REDIS_PASSWORD=

# Milvus Credentials
MILVUS_HOST=milvus_db
MILVUS_PORT=19530

# Recommendation Engine Configuration
LAST_UPDATE_CUT=24 months
SIMILARITY_LIST_SIZE=200
```

#### Step 3: The `docker-compose.yml` File

This file is the heart of your local setup. It defines every container, its ports, environment variables, and how they connect.

```yaml
# github-trending/docker-compose.yml

services:
  # --- INFRASTRUCTURE ---

  rabbitmq:
    image: rabbitmq:management-alpine
    container_name: rabbitmq
    ports:
      - "5672:5672"    # AMQP port for services
      - "15672:15672"  # Management UI
    volumes:
      - ./data/rabbitmq:/var/lib/rabbitmq/
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    env_file:
      - ./.env

  postgres:
    image: postgres:17.5-alpine
    container_name: postgres_db
    env_file:
      - ./.env
    ports:
      - "5432:5432"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data/
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d github_meta"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: minio_storage
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"  # API port
      - "9001:9001"  # Console UI
    env_file:
      - ./.env
    volumes:
      - ./data/minio:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    container_name: clickhouse_db
    ports:
      - "8123:8123" # HTTP interface
      - "9009:9000" # TCP port for native client
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    volumes:
      - ./data/clickhouse:/var/lib/clickhouse
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8123/ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    env_file:
      - ./.env

  redis:
    image: redis:alpine
    container_name: redis_cache
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  milvus:
    image: milvusdb/milvus:v2.2.0-standalone
    container_name: milvus_db
    ports:
      - "19530:19530"
    volumes:
      - milvus_data:/milvus/data

  # --- APPLICATION SERVICES (Golang) ---

  discovery:
    build:
      context: .
      dockerfile: ./cmd/discovery/Dockerfile
    container_name: discovery
    depends_on:
      rabbitmq: { condition: service_healthy }
      postgres: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env
    
  scheduler:
    build:
      context: .
      dockerfile: ./cmd/scheduler/Dockerfile
    container_name: scheduler
    depends_on:
      rabbitmq: { condition: service_healthy }
      postgres: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env

  crawler:
    build:
      context: .
      dockerfile: ./cmd/crawler/Dockerfile
    # Let's say we want to run 3 crawler instances
    deploy:
      replicas: 3
    depends_on:
      rabbitmq: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env
      
  processor:
    build:
      context: .
      dockerfile: ./cmd/processor/Dockerfile
    container_name: processor
    depends_on:
      rabbitmq: { condition: service_healthy }
      postgres: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env

  api-server:
    build:
      context: .
      dockerfile: ./cmd/api/Dockerfile
    container_name: api-server
    ports:
      - "8080:8080"
    depends_on:
      rabbitmq: { condition: service_healthy }
      postgres: { condition: service_healthy }
      clickhouse: { condition: service_healthy }
      minio: { condition: service_healthy }
      redis: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env

  embedding-service:
    build:
      context: .
      dockerfile: ./cmd/embedding-service/Dockerfile
    container_name: embedding_service
    depends_on:
      rabbitmq: { condition: service_healthy }
      postgres: { condition: service_healthy }
      minio: { condition: service_healthy }
      milvus: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env

  similarity-engine-service:
    build:
      context: .
      dockerfile: ./cmd/similarity-engine-service/Dockerfile
    container_name: similarity_engine_service
    depends_on:
      postgres: { condition: service_healthy }
      milvus: { condition: service_healthy }
      redis: { condition: service_healthy }
    restart: unless-stopped
    env_file:
      - ./.env

volumes:
  rabbitmq_data:
  postgres_data:
  minio_data:
  clickhouse_data:
  redis_data:
  milvus_data:
```

#### Step 4: Generic Go `Dockerfile`

Inside each of your service directories (`crawler-service`, `api-server`, etc.), you will need a `Dockerfile`. We can use a single, efficient, multi-stage template for all of them.

Create this file in `github-trending/cmd/<service>/Dockerfile` (e.g., `github-trending/cmd/crawler/Dockerfile`):

```dockerfile
# github-trending/cmd/<service>/Dockerfile

# --- Builder Stage ---
# Use the official Go image as a builder.
FROM golang:latest AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the internal shared code.
COPY internal ./internal

# Copy the rest of the application source code.
COPY cmd/<service> .

# Build the Go application.
# -o /app/main specifies the output file.
# CGO_ENABLED=0 is important for creating a static binary for Alpine.
# -ldflags "-s -w" strips debug symbols to make the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o /app/main .

# --- Final Stage ---
# Use a minimal Alpine image for the final container.
FROM alpine:latest

# Set the working directory.
WORKDIR /root/

# Copy the built binary from the builder stage.
COPY --from=builder /app/main .

# (Optional) Copy any config files if needed.
# COPY config.yml .

# Command to run the application.
CMD ["./main"]
```

You will place a copy of this `Dockerfile` into each of the five Go service directories, replacing `<service>` with the actual service name (e.g., `api`, `crawler`, `discovery`, `processor`, `scheduler`).

### Part 3: Go Code Design & Project Structure

We will use a **monorepo** approach. All your Go code will live in the `github-trending` directory. This is simpler to manage for a new project. We will leverage Go's module system.

#### Step 1: Initialize Go Module

In the root of your project, run:

```bash
go mod init github.com/your-username/github-trending
```

#### Step 2: High-Level Code Structure

Your project layout will look like this:

```
github-trending/
├── .env
├── docker-compose.yml
├── go.mod
├── go.sum
├── cmd/
│   ├── api/
│   │   ├── Dockerfile
│   │   └── main.go
│   ├── crawler/
│   │   ├── Dockerfile
│   │   └── main.go
│   ├── discovery/
│   │   ├── Dockerfile
│   │   └── main.go
│   ├── embedding-service/
│   │   ├── Dockerfile
│   │   └── main.go
│   ├── processor/
│   │   ├── Dockerfile
│   │   └── main.go
│   ├── scheduler/
│   │   ├── Dockerfile
│   │   └── main.go
│   └── similarity-engine-service/
│       ├── Dockerfile
│       └── main.go
├── internal/
│   ├── config/          # Load config from ENV vars
│   │   └── config.go
│   ├── database/
│   │   ├── clickhouse.go
│   │   ├── milvus.go
│   │   ├── minio.go
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── github/          # Your GitHub API client wrapper
│   │   └── client.go
│   ├── messaging/       # RabbitMQ logic
│   │   └── rabbitmq.go
│   └── models/          # Your core data structs
│       └── repository.go
└── storage/
    ├── postgres/
    │   └── schema.sql
    └── clickhouse/
        └── schema.sql
```

**Explanation:**

*   **`cmd/`**: This is the standard Go layout for applications. Each subdirectory is a self-contained microservice with its own `main.go` entrypoint.
*   **`internal/`**: This is for shared code that is *internal* to your project. Go's tooling prevents other projects from importing packages from an `internal` directory. This is perfect for our shared database clients, models, and messaging logic.
    *   **`config`**: A package to read configuration (like `RABBITMQ_URL`) from environment variables.
    *   **`database`**: Contains functions to connect to and query PostgreSQL, ClickHouse, Redis, MinIO, and Milvus.
    *   **`github`**: You should build your own small wrapper around the GitHub API client. This lets you centralize rate limiting logic, error handling, and authentication.
    *   **`messaging`**: Functions to connect to RabbitMQ, publish messages to a queue, and consume messages from a queue.
    *   **`models`**: The definition of your core data types (e.g., `RepositoryStats`, `RepositoryMeta`).
*   **`storage/`**: Contains the database schema files.

#### Step 3: Example Code Stub (`api-server/main.go`)

Let's write a skeleton for the `api-server` to show how it all connects.

```go
// github-trending/cmd/api/main.go
package main

import (
	"log"

	"github.com/teomiscia/github-trending/internal/api"
	"github.com/teomiscia/github-trending/internal/config"
	"github.com/teomiscia/github-trending/internal/database"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Connect to Redis
	redisClient, err := database.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 3. Connect to PostgreSQL
	postgresConnection, err := database.NewPostgresConnection(cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	// 4. Create and start the API server
	server := api.NewServer(redisClient, postgresConnection)

	log.Println("API Server started. Listening on :8080")
	log.Fatal(server.Run(":8080"))
}
```

#### Step 4: Example Code Stub (`internal/api/api.go`)

```go
package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/database"
)

func NewServer(redisClient *redis.Client, db *database.PostgresConnection) *gin.Engine {
	router := gin.Default()

	router.GET("/retrieveList", handleRetrieveList(redisClient, db))
	router.POST("/trackOpenRepository", handleTrackOpenRepository(db))

	return router
}

func handleRetrieveList(redisClient *redis.Client, db *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("sessionId")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		languages := strings.Split(c.Query("languages"), ",")
		if len(languages) == 1 && languages[0] == "" {
			languages = []string{}
		}
		tags := strings.Split(c.Query("tags"), ",")
		if len(tags) == 1 && tags[0] == "" {
			tags = []string{}
		}

		var recommendedRepoIDs []int64

		// 1. Query the repository_views table for user history
		userHistoryRepoIDs, err := db.GetRecentClickedRepositoryIDs(sessionID, 15) // Get last 15 clicked repos
		if err != nil {
			log.Printf("Failed to get user history from Postgres: %v", err)
			// Fallback to generic trending if history fails
			repoIDs, err := db.GetTrendingRepositoryIDs(languages, tags)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve repository list"})
				return
			}
			recommendedRepoIDs = repoIDs
		} else if len(userHistoryRepoIDs) == 0 {
			// 2. If no user history exists, fall back to generic trending
			repoIDs, err := db.GetTrendingRepositoryIDs(languages, tags)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve repository list"})
				return
			}
			recommendedRepoIDs = repoIDs
		} else {
			// 3. If user history exists, aggregate similar repositories
			candidateScores := make(map[int64]float64)
			for _, historyRepoID := range userHistoryRepoIDs {
				redisKey := fmt.Sprintf("similar:%d", historyRepoID)
				// Fetch top 50 similar repos for each history item
				similarRepos, err := redisClient.ZRevRangeWithScores(context.Background(), redisKey, 0, 49).Result()
				if err != nil {
					log.Printf("Failed to get similar repos from Redis for %d: %v", historyRepoID, err)
					continue
				}
				for _, z := range similarRepos {
					repoID, _ := strconv.ParseInt(fmt.Sprintf("%.0f", z.Member), 10, 64)
					candidateScores[repoID] += z.Score
				}
			}

			// Convert map to slice for sorting
			type scoredRepo struct {
				ID    int64
				Score float64
			}
			var scoredRepos []scoredRepo
			for id, score := range candidateScores {
				scoredRepos = append(scoredRepos, scoredRepo{ID: id, Score: score})
			}

			// Sort by score in descending order
			sort.Slice(scoredRepos, func(i, j int) bool {
				return scoredRepos[i].Score > scoredRepos[j].Score
			})

			for _, sr := range scoredRepos {
				recommendedRepoIDs = append(recommendedRepoIDs, sr.ID)
			}
		}

		// Filter out repositories that have already been seen in this session
		seenRepoIDs, err := redisClient.SMembers(context.Background(), sessionID).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve seen repositories"})
			return
		}

		seenRepoIDMap := make(map[int64]bool)
		for _, idStr := range seenRepoIDs {
			id, _ := strconv.ParseInt(idStr, 10, 64)
			seenRepoIDMap[id] = true
		}

		var finalRepoIDs []int64
		for _, id := range recommendedRepoIDs {
			if !seenRepoIDMap[id] {
				finalRepoIDs = append(finalRepoIDs, id)
			}
		}

		// Paginate the results
		page := 0
		if pageStr := c.Query("page"); pageStr != "" {
			page, _ = strconv.Atoi(pageStr)
		}
		pageSize := 50
		start := page * pageSize
		end := start + pageSize
		if start > len(finalRepoIDs) {
			finalRepoIDs = []int64{}
		} else if end > len(finalRepoIDs) {
			finalRepoIDs = finalRepoIDs[start:]
		} else {
			finalRepoIDs = finalRepoIDs[start:end]
		}

		// Add the new repositories to the seen set in Redis
		if len(finalRepoIDs) > 0 {
			var repoIDStrs []string
			for _, id := range finalRepoIDs {
				repoIDStrs = append(repoIDStrs, fmt.Sprint(id))
			}
			redisClient.SAdd(context.Background(), sessionID, repoIDStrs)
		}

		c.JSON(http.StatusOK, gin.H{
			"sessionId":    sessionID,
			"repositories": finalRepoIDs,
		})
	}
}

func handleTrackOpenRepository(db *database.PostgresConnection) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody struct {
			SessionID    string `json:"sessionId"`
			RepositoryID int64  `json:"repositoryId"`
		}

		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := db.TrackRepositoryView(requestBody.SessionID, requestBody.RepositoryID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track repository view"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	}
}


### Your First Steps: Putting It All Together

1.  **Create the Full Directory Structure:** Use the `mkdir` commands from above.
2.  **Create the `.env` File:** Populate it with your environment variables.
3.  **Create the Go Module:** Run `go mod init ...`.
4.  **Create the `Dockerfile`s:** Create the generic `Dockerfile` and place a copy in each of the five `cmd/*` directories. (e.g., in `cmd/crawler/Dockerfile`).
5.  **Write Skeletons:** Write a basic `main.go` for each service, just to prove they can start up. For now, they can just print a message like `"Crawler service starting..."`.
6.  **Launch!** From the root `github-trending` directory, run:

    ```bash
    docker-compose up --build
    ```

You will see Docker build each of your Go services and then start all the containers. You can visit `http://localhost:15672` (user: `guest`, pass: `guest`) to see the RabbitMQ management UI, `http://localhost:9001` for MinIO, and connect to the databases on their respective ports.

This setup gives you a complete, professional foundation. You can now focus on writing the Go logic inside each microservice, knowing that the infrastructure and communication layer is already handled.
