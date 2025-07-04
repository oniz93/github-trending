# GitHub Trending Microservices

This project implements a system of microservices designed to track and analyze trending GitHub repositories. It leverages a robust, one-way data flow to discover, crawl, process, and serve data related to popular open-source projects.

## Architecture Overview

The system is composed of five core microservices, each with a distinct responsibility:

-   **`discovery-service`**: Finds new repositories to track using the GitHub Search API.
-   **`scheduler-service`**: Schedules refreshes for already tracked repositories.
-   **`crawler-service`**: Fetches raw data from the GitHub API for specified repositories.
-   **`processor-service`**: Parses, cleans, and saves raw data to various databases.
-   **`api-server`**: Provides a public-facing API for querying trending data.

For a detailed explanation of the architecture and data flow, please refer to [GEMINI.md](./GEMINI.md).

## Local Development Setup

This project uses Docker Compose for a streamlined local development environment.

### Prerequisites

-   [Docker Desktop](https://www.docker.com/products/docker-desktop) (or Docker Engine and Docker Compose)
-   [Go](https://golang.org/dl/) (for developing the Go microservices)

### Setup Steps

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-username/github-trending.git
    cd github-trending
    ```

2.  **Initialize Go Module:**
    ```bash
    go mod tidy
    ```

3.  **Create `.env` file:**
    Create a `.env` file in the root of your project to store environment variables common to all services. An example is provided in [GEMINI.md](./GEMINI.md).

4.  **Build and Run Services:**
    The `docker-compose.yml` file defines all the necessary infrastructure (RabbitMQ, PostgreSQL, MinIO, ClickHouse, Redis) and the Go microservices. To build the Go services and start all containers, run:
    ```bash
    docker-compose up --build
    ```
    This command will build the Docker images for each Go service and then start all defined services in the foreground. You can add `-d` to run them in detached mode (background).

### Accessing Services and UIs

Once the services are running, you can access the following:

-   **RabbitMQ Management UI**: `http://localhost:15672` (User: `user`, Pass: `password`)
-   **MinIO Console UI**: `http://localhost:9001` (User: `minioadmin`, Pass: `minioadmin`)
-   **PostgreSQL**: Connect to `localhost:5432` (User: `user`, Password: `password`, Database: `github_meta`)
-   **ClickHouse**: HTTP interface at `http://localhost:8123`, Native client interface at `localhost:9009` (User: `clickhouse`, Password: `clickhouse`)
-   **Redis**: Connect to `localhost:6379`
-   **API Server**: `http://localhost:8080`

## API Endpoints

-   `GET /retrieveList?sessionId=<session_id>&languages=<language1>,<language2>&tags=<tag1>,<tag2>&page=<page>`: Retrieves a paginated list of trending repositories, filtered by languages and tags. If `sessionId` is not provided, a new session is created.
-   `POST /trackOpenRepository`: Tracks a user's click on a repository. The request body should be a JSON object with `sessionId` and `repositoryId` fields.

## Project Structure

The project follows a monorepo approach with a clear structure:

```
github-trending/
├── .env
├── docker-compose.yml
├── go.mod
├── go.sum
├── cmd/             # Contains main entry points for each microservice
│   ├── api/
│   ├── crawler/
│   ├── discovery/
│   ├── processor/
│   └── scheduler/
├── internal/        # Shared internal Go packages (config, database, github client, messaging, models)
│   ├── config/
│   ├── database/
│   ├── github/
│   ├── messaging/
│   └── models/
└── GEMINI.md        # Detailed architectural and setup documentation
```

For a more in-depth look at the project structure and Go code design, please refer to [GEMINI.md](./GEMINI.md).

## Contributing

(Optional: Add guidelines for contributing, e.g., code style, testing, pull request process.)
