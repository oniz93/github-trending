# 🚀 GitHub Trending & Discovery Engine

A high-performance, distributed system engineered for the discovery, deep analysis, and semantic search of GitHub repositories. This project leverages a microservices architecture and an event-driven pipeline to process large-scale repository metadata, generating insights through vector embeddings and real-time analytics.

## 🏗 System Architecture & Concurrency Model

The system is designed as a decoupled, event-driven pipeline where specialized services communicate via **RabbitMQ**. This architecture allows for independent scaling of each component and ensures high availability and fault tolerance.

*   **Distributed Ingestion**: The `discovery` and `crawler` services utilize worker pools to concurrently fetch and filter repository data from the GitHub API.
*   **Event-Driven Pipeline**: 
    1.  **Discovery**: Identifies new or trending repositories and publishes to `repos_to_crawl`.
    2.  **Crawler**: Pulls metadata, tags, and languages, then publishes to `raw_data_to_process`.
    3.  **Processor**: Downloads READMEs, stores them in the S3 object store (**SeaweedFS**), and triggers embedding generation via `readme_to_embed`.
    4.  **Writer**: Handles high-throughput writes to **PostgreSQL** and **ClickHouse**.
*   **High-Efficiency Storage**: 
    *   **ClickHouse Integration**: Implements a custom **row-hashing mechanism** in Go to deduplicate time-series metric inserts, significantly reducing storage overhead while maintaining a granular history of repository growth.
    *   **Qdrant Vector Search**: Powers a semantic similarity engine, enabling "find similar repos" functionality based on content embeddings rather than simple keyword matching.

## 🛠 Technical Stack

### Backend & Microservices
*   **Language**: **Go (1.23+)** – Core logic, high-performance concurrency, and microservices.
*   **API Framework**: **Gin** – RESTful API entry point.
*   **Messaging**: **RabbitMQ** – Distributed event bus for service orchestration.
*   **AI/Embeddings**: **Python (FastAPI)** – Specialized services for NLP and vector generation.

### Data & Infrastructure
*   **Analytical DB**: **ClickHouse** – Real-time trend analysis and growth tracking.
*   **Primary DB**: **PostgreSQL** – Relational storage for repository metadata.
*   **Vector DB**: **Qdrant** – AI-powered semantic similarity search.
*   **Caching**: **Redis** – Low-latency data retrieval for the API.
*   **Object Storage**: **SeaweedFS** – S3-compatible storage for READMEs and assets (any S3-compatible store works via `S3_ENDPOINT`).
*   **Observability**: **OpenTelemetry** – Distributed tracing and system telemetry.

## 📂 Microservices Overview

*   `cmd/api`: High-performance REST API.
*   `cmd/crawler`: Concurrent metadata and content scraper.
*   `cmd/processor`: Orchestrates data transformation and storage triggers.
*   `cmd/embedding-service`: Generates vector embeddings for repository content.
*   `cmd/similarity-engine-service`: Calculates semantic similarity scores.
*   `cmd/social-poster`: Automated social media engagement (Twitter/X).
*   `cmd/og-image-service`: Dynamic Open Graph image generation.
*   `cmd/writer-service`: Batch-optimized data persistence.

## ⚡️ Getting Started

### Prerequisites
*   Docker & Docker Compose
*   Go 1.23+

### Installation & Deployment

1.  **Clone & Configure**:
    ```bash
    git clone https://github.com/teomiscia/github-trending.git
    cp .env.example .env # Configure your credentials
    ```

2.  **Orchestrate with Docker**:
    ```bash
    docker-compose up -d --build
    ```

The entire stack is containerized and orchestrated via Docker Compose, including all databases, message brokers, and Go/Python microservices.

## 📖 Key API Endpoints
*   `GET /repos/trending`: Retrieve growth-indexed trending repositories.
*   `GET /repos/{id}/similar`: Semantic similarity search via Qdrant.

