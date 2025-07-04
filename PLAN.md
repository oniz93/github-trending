# Recommendation Engine Implementation Plan (Method C: Embeddings)

This document outlines the steps to build and integrate a sophisticated, real-time recommendation engine into the existing microservices architecture. The goal is to provide users with personalized repository recommendations based on their interaction history, similar to modern content discovery platforms.

We will use a state-of-the-art **Embeddings + Vector Search** approach.

---

### **Phase 1: Infrastructure Expansion**

The new services require an addition to our core infrastructure: a vector database.

*   **Step 1.1: Add Vector Database to Docker Compose**
    *   **Action:** Add a new service definition for **Milvus** (a popular open-source vector database) to the `docker-compose.yml` file.
    *   **Details:** This service will include the Milvus standalone container, its persistent volume, and necessary environment variables. It will be configured to communicate with the other services.

*   **Step 1.2: Add New Microservices to Docker Compose**
    *   **Action:** Add two new Go services, `embedding-service` and `similarity-engine-service`, to the `docker-compose.yml` file.
    *   **Details:** These definitions will mirror the existing services, including build context, Dockerfile paths, dependencies on other services (Postgres, MinIO, Milvus, Redis), and environment file configuration.

*   **Step 1.3: Create New Service Directories**
    *   **Action:** Create the directory structure for the new services.
    *   **Commands:**
        ```bash
        mkdir -p cmd/embedding-service
        mkdir -p cmd/similarity-engine-service
        touch cmd/embedding-service/main.go
        touch cmd/embedding-service/Dockerfile
        touch cmd/similarity-engine-service/main.go
        touch cmd/similarity-engine-service/Dockerfile
        ```
    *   **Details:** The Dockerfiles will be copies of the generic Go Dockerfile used by the other services.

*   **Step 1.4: Update Configuration**
    *   **Action:** Add new configuration variables to the `.env` file.
    *   **Variables:**
        *   `MILVUS_HOST`, `MILVUS_PORT`
        *   `LAST_UPDATE_CUT`: A string representing the time cutoff for processing repositories (e.g., "24 months").
        *   `SIMILARITY_LIST_SIZE`: The number of similar items to store for each repository (e.g., "200").

---

### **Phase 2: `embedding-service` - Understanding Content**

This service's sole responsibility is to convert repository READMEs into semantic vectors (embeddings).

*   **Step 2.1: Implement a MinIO Webhook Trigger**
    *   **Action:** The `processor-service` will be modified. After successfully saving a `README.md` file to MinIO, it will publish a message to a new RabbitMQ queue, `readme_to_embed`.
    *   **Message Body:** `{"repository_id": 12345, "minio_path": "/path/to/readme.md"}`.

*   **Step 2.2: Build the Embedding Consumer**
    *   **Action:** The `embedding-service` will consume messages from the `readme_to_embed` queue.
    *   **Logic:**
        1.  On receiving a message, query PostgreSQL for the repository's metadata, specifically its `pushed_at` date.
        2.  Check if `pushed_at` is within the `LAST_UPDATE_CUT` period. If not, acknowledge the message and stop.
        3.  Attempt to fetch the `README` content from MinIO using the path from the message.
        4.  **Fallback:** If the file is not in MinIO, query the repository's `readme_url` from PostgreSQL and download the content directly.
        5.  Instantiate a sentence-transformer model (using a Go-native library).
        6.  Generate a vector embedding from the `README` content.
        7.  Connect to Milvus and save the embedding vector, using the `repository_id` as its key.
        8.  Acknowledge the RabbitMQ message.

---

### **Phase 3: `similarity-engine-service` - Pre-calculating Similarity**

This service builds the fast-access similarity lists that the API will use.

*   **Step 3.1: Implement Scheduled Execution**
    *   **Action:** The `similarity-engine-service` will not be a queue consumer. It will run as a scheduled batch job (e.g., using a `time.Ticker` in Go to run every 6 hours).

*   **Step 3.2: Build the Similarity List Generator**
    *   **Action:** On its scheduled interval, the service will execute the following logic.
    *   **Logic:**
        1.  Fetch all repository IDs from PostgreSQL that have been updated within the `LAST_UPDATE_CUT` period.
        2.  For each `repository_id`, query Milvus to get its embedding vector.
        3.  Perform a similarity search in Milvus using that vector to find the top `N` nearest neighbors (where `N` is `SIMILARITY_LIST_SIZE`).
        4.  Connect to Redis and store this result in a **Sorted Set**.
        5.  **Redis Schema:**
            *   **Key:** `similar:repo_id` (e.g., `similar:12345`)
            *   **Members:** The IDs of the similar repositories.
            *   **Score:** The similarity score (e.g., cosine similarity) returned by Milvus. This allows us to fetch the most similar items easily.

---

### **Phase 4: API Integration - Serving Personalized Content**

This phase modifies the `api-server` to use the pre-computed lists.

*   **Step 4.1: Enhance the `retrieveList` Endpoint**
    *   **Action:** The logic within the `handleRetrieveList` function in `api.go` will be significantly updated.
    *   **New Logic Flow:**
        1.  Get the `sessionId` from the request.
        2.  Query the `repository_views` table in PostgreSQL to get a list of the last 10-15 repository IDs the user has clicked on (the "user history").
        3.  **If no user history exists (new user):** Fall back to the existing logic of showing a generic list of trending repositories.
        4.  **If user history exists:**
            a. For each repository ID in the user history, fetch the pre-computed list of similar repositories from its Redis sorted set (e.g., `ZREVRANGE similar:12345 0 50`).
            b. Aggregate these lists into a single candidate pool. Use the similarity scores to rank candidates. A simple approach is to sum the scores for repositories that appear in multiple lists.
            c. Filter out any repositories that are already in the user's click history.
            d. Filter out any repositories that have already been served in the current session (using the existing Redis set for the `sessionId`).
            e. Serve the top 50 repositories from the final, ranked, and filtered list.

*   **Step 4.2: Update Database Access**
    *   **Action:** Create a new function in `internal/database/postgres.go` named `GetRecentClickedRepositoryIDs(sessionID string, limit int)`.

---

### **Phase 5: Documentation**

*   **Step 5.1: Update `README.md` and `GEMINI.md`**
    *   **Action:** Update both documents to describe the new `embedding-service` and `similarity-engine-service`, the addition of Milvus to the stack, and the new, personalized functionality of the `api-server`.
