This document provides a comprehensive technical blueprint of the GitHub Trending project. It is intended for a Large Language Model (LLM) to understand the architecture, code, and deployment of the system.

### Part 1: Core Architecture & Services

The system is a microservices-based application designed to discover, analyze, and serve trending GitHub repositories.

**Services:**

*   **`discovery-service` (Go):** Periodically searches the GitHub API for new repositories based on criteria like star count. Publishes found repositories to the `repos_to_crawl` queue.
*   **`scheduler-service` (Go):** Periodically queries the database for existing repositories that need to be refreshed and publishes them to the `repos_to_crawl` queue.
*   **`crawler-service` (Go):** Consumes repository information from the `repos_to_crawl` queue. Fetches detailed data for each repository from the GitHub API (stats, languages, tags, etc.) and publishes the raw data to the `raw_data_to_process` queue.
*   **`processor-service` (Go):** Consumes from the `raw_data_to_process` queue. It processes the raw data, and then publishes messages to two queues: `repos_to_write` (for database storage) and `readme_to_embed` (for embedding generation).
*   **`writer-service` (Go):** Consumes from the `repos_to_write` queue and writes the processed repository data to the PostgreSQL and ClickHouse databases.
*   **`embedding-api-service` (Python/FastAPI):** A standalone API that exposes an endpoint (`/embed`) to generate sentence embeddings for a given text using a pre-trained SentenceTransformer model.
*   **`embedding-autoscaler` (Go):** A smart proxy that sits in front of the `embedding-api-service`. It dynamically scales the number of `embedding-api-service` instances from 0 to a configured maximum based on request load. It also load-balances requests among the running instances.
*   **`embedding-service` (Go):** Consumes from the `readme_to_embed` queue. It fetches the README content from MinIO, calls the `embedding-api-service` (via the autoscaler) to get the embedding, and stores the resulting vector in the Qdrant vector database.
*   **`similarity-engine-service` (Go):** Periodically calculates similarity scores between repositories. It fetches embeddings from Qdrant, computes similarity, and stores the results in Redis for fast access.
*   **`api-server` (Go):** The public-facing API for the application. It handles user requests, queries the databases (PostgreSQL, ClickHouse, Redis) to get trending and personalized repository data, and returns the results as JSON.
*   **`web` (React Native/Expo):** A mobile and web application that provides a user interface for browsing trending repositories. It features a TikTok-style vertical scrolling feed and a detailed view with a rendered README.

**Data Flow:**

1.  `Discovery/Scheduler` -> `repos_to_crawl` (RabbitMQ)
2.  `repos_to_crawl` -> `Crawler`
3.  `Crawler` -> `raw_data_to_process` (RabbitMQ)
4.  `raw_data_to_process` -> `Processor`
5.  `Processor` -> `repos_to_write` (RabbitMQ)
6.  `repos_to_write` -> `Writer Service` -> `PostgreSQL` & `ClickHouse`
7.  `Processor` -> `readme_to_embed` (RabbitMQ)
8.  `readme_to_embed` -> `Embedding Service` -> `MinIO` & `Embedding API` -> `Qdrant`
9.  `Similarity Engine` -> `Qdrant` & `PostgreSQL` -> `Redis`
10. `API Server` -> `PostgreSQL`, `ClickHouse`, `Redis` -> `web` (User)

### Part 2: Local Development & Deployment (Docker Swarm)

The application is deployed as a Docker Swarm stack.

**`.env` File:**

```dotenv
RABBITMQ_URL=amqp://user:password@rabbitmq:5672/
GITHUB_TOKEN=<your_github_token>
GITHUB_CALLBACK_SECRET=<your_github_callback_secret>

# RabbitMQ Credentials
RABBITMQ_DEFAULT_USER=user
RABBITMQ_DEFAULT_PASS=password
RABBITMQ_ERLANG_COOKIE="<a_secure_random_string>"

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

# Qdrant Credentials
QDRANT_HOST=qdrant_db
QDRANT_PORT=6333

# Recommendation Engine Configuration
LAST_UPDATE_CUT=12 months
SIMILARITY_LIST_SIZE=200

# Autoscaler configuration
EMBEDDING_API_MAX_INSTANCES=3
EMBEDDING_API_IDLE_TIMEOUT=600
```

**`docker-compose.yml`:**

```yaml
# github-trending/docker-compose.yml

services:
  # --- INFRASTRUCTURE ---

  rabbitmq:
    image: rabbitmq:management-alpine
    container_name: rabbitmq
    ports:
      - "5672:5672"
      - "15672:15672"
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
    image: postgres:17.6-alpine
    container_name: postgres_db
    env_file:
      - ./.env
    ports:
      - "5432:5432"
    volumes:
      - ./data/postgres:/var/lib/postgresql/data/
      - ./postgres-init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d github_meta"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: postgres -c shared_preload_libraries=pg_stat_statements

  minio:
    image: minio/minio:latest
    container_name: minio_storage
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
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
      - "8123:8123"
      - "9009:9000"
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
      - ./data/redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  qdrant:
    image: qdrant/qdrant:latest
    container_name: qdrant_db
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - ./data/qdrant:/qdrant/storage

  # --- APPLICATION SERVICES ---

  discovery:
    build:
      context: .
      dockerfile: ./cmd/discovery/Dockerfile
    image: github-trending/discovery
    depends_on:
      - rabbitmq
      - postgres
    env_file:
      - ./.env

  crawler:
    build:
      context: .
      dockerfile: ./cmd/crawler/Dockerfile
    image: github-trending/crawler
    deploy:
      replicas: 6
    depends_on:
      - rabbitmq
    env_file:
      - ./.env

  processor:
    build:
      context: .
      dockerfile: ./cmd/processor/Dockerfile
    image: github-trending/processor
    deploy:
      replicas: 3
    depends_on:
      - rabbitmq
      - minio
    env_file:
      - ./.env

  writer-service:
    build:
      context: .
      dockerfile: ./cmd/writer-service/Dockerfile
    image: github-trending/writer-service
    depends_on:
      - rabbitmq
      - postgres
      - clickhouse
    env_file:
      - ./.env

  api-server:
    build:
      context: .
      dockerfile: ./cmd/api/Dockerfile
    image: github-trending/api-server
    ports:
      - "8080:8080"
    depends_on:
      - rabbitmq
      - postgres
      - clickhouse
      - minio
      - redis
    env_file:
      - ./.env

  embedding-service:
    build:
      context: .
      dockerfile: ./cmd/embedding-service/Dockerfile
    image: github-trending/embedding-service
    depends_on:
      - rabbitmq
      - postgres
      - minio
      - qdrant
    env_file:
      - ./.env

  embedding-api-service:
    build:
      context: .
      dockerfile: ./cmd/embedding-autoscaler/Dockerfile
    image: github-trending/embedding-autoscaler
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    env_file:
      - ./.env

  embedding-api-instance:
    build:
      context: ./cmd/embedding-api-service
    image: github-trending/embedding-api-service
    deploy:
      replicas: 0
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  similarity-engine-service:
    build:
      context: .
      dockerfile: ./cmd/similarity-engine-service/Dockerfile
    image: github-trending/similarity-engine-service
    depends_on:
      - postgres
      - qdrant
      - redis
    env_file:
      - ./.env
```

**Deployment Workflow:**

1.  **Initialize Swarm:** `docker swarm init`
2.  **Build images:** `docker compose build`
3.  **Deploy stack:** `docker stack deploy -c docker-compose.yml github-trending`

### Part 3: Project Structure

```
github-trending/
├── .env
├── docker-compose.yml
├── go.mod
├── go.sum
├── web/
│   ├── app/
│   ├── components/
│   ├── services/
│   └── ...
├── cmd/
│   ├── api/
│   ├── crawler/
│   ├── discovery/
│   ├── embedding-api-service/
│   ├── embedding-autoscaler/
│   ├── embedding-service/
│   ├── processor/
│   ├── scheduler/
│   ├── similarity-engine-service/
│   └── writer-service/
├── internal/
│   ├── api/
│   ├── config/
│   ├── database/
│   ├── github/
│   ├── messaging/
│   └── models/
└── storage/
    ├── postgres/
    │   └── schema.sql
    └── clickhouse/
        └── schema.sql
```

### Part 4: Database Schemas

**PostgreSQL (`storage/postgres/schema.sql`):**

```sql
CREATE TABLE IF NOT EXISTS owners (
    id BIGINT PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    avatar_url VARCHAR(255),
    html_url VARCHAR(255),
    type VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS licenses (
    key VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    spdx_id VARCHAR(255),
    url VARCHAR(255),
    node_id VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS repositories (
    id BIGINT PRIMARY KEY,
    node_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) UNIQUE NOT NULL,
    owner_id BIGINT REFERENCES owners(id),
    description TEXT,
    html_url VARCHAR(255),
    homepage VARCHAR(255),
    default_branch VARCHAR(255),
    license_key VARCHAR(255) REFERENCES licenses(key),
    readme_url VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE,
    is_fork BOOLEAN,
    is_template BOOLEAN,
    is_archived BOOLEAN,
    is_disabled BOOLEAN,
    last_crawled_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS languages (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_languages (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    language_id INT REFERENCES languages(id) ON DELETE CASCADE,
    size BIGINT NOT NULL,
    PRIMARY KEY (repository_id, language_id)
);

CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_tags (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (repository_id, tag_id)
);

CREATE TABLE IF NOT EXISTS topics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_topics (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    topic_id INT REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (repository_id, topic_id)
);

CREATE TABLE IF NOT EXISTS repository_views (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    repository_id BIGINT NOT NULL,
    viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS repository_similarity (
    id BIGINT PRIMARY KEY,
    data JSONB NOT NULL
);
```

**ClickHouse (`storage/clickhouse/schema.sql`):**

```sql
CREATE TABLE IF NOT EXISTS repository_stats (
    event_date Date,
    event_time DateTime,
    repository_id UInt64,
    stargazers_count UInt64,
    watchers_count UInt64,
    forks_count UInt64,
    open_issues_count UInt64,
    pushed_at DateTime,
    score Float64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (repository_id, event_time);
```

### Part 5: Frontend (React Native / Expo)

**Technology Stack:**

*   **Framework:** React Native with Expo
*   **Language:** TypeScript
*   **Navigation:** React Navigation
*   **Gestures:** React Native Gesture Handler, React Native Reanimated
*   **API Client:** Axios
*   **Web Rendering:** React Native WebView (for native), iframe (for web)

**Project Structure (`web/` directory):

```
web/
├── app/              # Screens and navigation
│   ├── _layout.tsx
│   └── index.tsx
├── components/       # Reusable components
│   ├── RepositoryCard.tsx
│   └── TopBar.tsx
├── services/         # API services
│   └── api.ts
├── types/            # TypeScript types
│   └── repository.ts
├── hooks/
└── assets/
```

**Development Workflow:**

1.  **Navigate to the `web` directory:**
    ```bash
    cd web
    ```
2.  **Install dependencies:**
    ```bash
    npm install
    ```
3.  **Start the development server:**
    ```bash
    # For web
    npm run web

    # For iOS
    npm run ios

    # For Android
    npm run android
    ```

### Part 6: Key Code Snippets

**Autoscaler (`cmd/embedding-autoscaler/main.go`):**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
)

// ... (full source code of autoscaler) ...
```