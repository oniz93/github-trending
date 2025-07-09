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
                                                                                                `Processor` -> `Embeddings Queue` -> `Embedding Service` -> `Milvus`
                                                                                                `Similarity Engine` -> `Milvus` -> `Redis`

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

## Summary of Changes (2025-07-08)

*   **Refactored Database Schema:** Separated repository metadata (stored in PostgreSQL) from time-series statistics (stored in ClickHouse) to improve performance and scalability.
*   **Fixed Data Ingestion Pipeline:** Corrected several bugs in the data ingestion pipeline that were causing errors and preventing data from being processed correctly.
*   **Improved Configuration Handling:** Updated the configuration loading logic to correctly parse custom duration units.
*   **Updated Documentation:** Updated the `GEMINI.md` and `README.md` files to reflect the latest architectural changes and fixes.