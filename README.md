# GitHub Trending

This project is a microservices-based application that discovers, analyzes, and displays trending repositories on GitHub.

## Architecture

The application is composed of the following microservices:

*   **`discovery-service`**: Finds new repositories to track.
*   **`scheduler-service`**: Schedules refreshes for repositories that are already being tracked.
*   **`crawler-service`**: Fetches repository data from the GitHub API.
*   **`processor-service`**: Processes and stores repository data in the appropriate databases.
*   **`embedding-service`**: Generates and stores semantic embeddings for repository READMEs.
*   **`similarity-engine-service`**: Calculates and stores similarity scores between repositories.
*   **`api-server`**: Provides a public API for accessing trending repository data.

This design creates a robust, one-way data flow:

`Discovery/Scheduler` -> `Crawl Queue` -> `Crawler` -> `Process Queue` -> `Processor` -> `Databases` <- `API Server`
                                                                                                `Processor` -> `Embeddings Queue` -> `Embedding Service` -> `Qdrant`
                                                                                                `Similarity Engine` -> `Qdrant` -> `Redis`

## Getting Started

To get started, you will need to have Docker and Docker Compose installed.

1.  Clone the repository.
2.  Create a `.env` file in the root of the project and populate it with your environment variables. You can use the `.env.example` file as a template.
3.  Run the following command to build and start the application:

    ```bash
    docker-compose up --build -d
    ```

## Project Structure

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
│   │   ├── qdrant.go
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
    *   **`database`**: Contains functions to connect to and query PostgreSQL, ClickHouse, Redis, MinIO, and Qdrant.
    *   **`github`**: You should build your own small wrapper around the GitHub API client. This lets you centralize rate limiting logic, error handling, and authentication.
    *   **`messaging`**: Functions to connect to RabbitMQ, publish messages to a queue, and consume messages from a queue.
    *   **`models`**: The definition of your core data types (e.g., `RepositoryStats`, `RepositoryMeta`).
*   **`storage/`**: Contains the database schema files.

## Summary of Changes (2025-07-09)

*   **Replaced Milvus with Qdrant:** The vector database has been changed from Milvus to Qdrant for storing README embeddings.
*   **Added `similarity-engine-service`:** This service calculates and stores similarity scores between repositories.
*   **Enhanced API Server:** The API server now provides personalized recommendations based on user history and uses Redis for caching.
*   **Improved Discovery Service:** The discovery service now uses a lock file to resume discovery and a trigger file to start discovery immediately.
*   **Improved Crawler Service:** The crawler service now checks if a repository has been updated before crawling it.
*   **Refactored Processor Service:** The processor service has been refactored into its own package.
*   **Updated Configuration Handling:** The configuration loading logic now correctly parses custom duration units for "months" and "years".
*   **Updated Data Models:** The data models have been updated to include more fields and new message structs have been added.
*   **Improved GitHub Client:** The GitHub client now includes a backoff mechanism to handle rate limiting.