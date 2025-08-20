# GitHub Trending

This project is a microservices-based application that discovers, analyzes, and displays trending repositories on GitHub.

## Architecture

The application is composed of the following microservices:

*   **`discovery-service`**: Finds new repositories to track.
*   **`scheduler-service`**: Schedules refreshes for repositories that are already being tracked.
*   **`crawler-service`**: Fetches repository data from the GitHub API.
*   **`processor-service`**: Processes and stores repository data in the appropriate databases.
*   **`writer-service`**: Writes repository data to PostgreSQL and ClickHouse.
*   **`embedding-api-service`**: A Python service that provides an API to generate text embeddings.
*   **`embedding-autoscaler`**: A Go service that automatically scales the `embedding-api-service` based on load.
*   **`embedding-service`**: Generates and stores semantic embeddings for repository READMEs.
*   **`similarity-engine-service`**: Calculates and stores similarity scores between repositories.
*   **`api-server`**: Provides a public API for accessing trending repository data.

This design creates a robust, one-way data flow:

`Discovery/Scheduler` -> `Crawl Queue` -> `Crawler` -> `Process Queue` -> `Processor` -> `Write Queue` -> `Writer Service` -> `Databases`
                                                                                                  `Processor` -> `Embeddings Queue` -> `Embedding Service` -> `Embedding API` -> `Qdrant`
                                                                                                  `Similarity Engine` -> `Qdrant` -> `Redis`

## Deployment (Docker Swarm)

This application is designed to be deployed as a Docker Swarm stack. This allows for easy scaling and management of the services.

### Prerequisites

- Docker Engine must be running in Swarm mode. If not, initialize it with:
  ```bash
  docker swarm init
  ```

### Initial Deployment

1.  **Build all service images:**
    ```bash
    docker compose build
    ```

2.  **Deploy the stack:**
    ```bash
    docker stack deploy -c docker-compose.yml github-trending
    ```

### Development Workflow

When you make changes to a service's code, follow these steps to update the running application:

1.  **Rebuild the image for the service you changed:**
    ```bash
    # Rebuild a specific service (e.g., api-server)
    docker compose build api-server

    # Or, to rebuild all services
    docker compose build
    ```

2.  **Update the running stack:**
    This command will perform a rolling update for the services with new images.
    ```bash
    docker stack deploy -c docker-compose.yml github-trending
    ```

### Managing the Stack

-   **List all services in the stack:**
    ```bash
    docker stack services github-trending
    ```

-   **View logs for a specific service:**
    ```bash
    docker service logs github-trending_api-server
    ```

-   **Restart a single service:**
    ```bash
    docker service update --force github-trending_api-server
    ```

-   **Remove the entire stack:**
    ```bash
    docker stack rm github-trending
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
