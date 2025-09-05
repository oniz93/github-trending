# Server Migration Plan

This document outlines the step-by-step process for migrating the GitHub Trending application to a new ARM64 VPS.

## 1. New Server Preparation

1.  **Provision VPS:** Start a new ARM64 VPS with 4 CPUs, 8GB RAM, and at least 30GB of storage (based on current usage of 15GB, with room to grow).
2.  **Install OS:** Install **Ubuntu Server 24.04 LTS** or the latest stable version.
3.  **Initial Server Setup:**
    *   Connect to the server via SSH.
    *   Update all system packages:
        ```bash
        sudo apt update && sudo apt upgrade -y
        ```
    *   Create a new user and grant it sudo privileges (replace `your_user` with your desired username):
        ```bash
        adduser your_user
        usermod -aG sudo your_user
        ```
    *   Log out and log back in as the new user.
4.  **Install Docker and Docker Compose:**
    *   Follow the official Docker documentation to install Docker Engine for Ubuntu. This is the recommended approach.
    *   Install the Docker Engine:
        ```bash
        sudo apt-get install ca-certificates curl
        sudo install -m 0755 -d /etc/apt/keyrings
        sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
        sudo chmod a+r /etc/apt/keyrings/docker.asc
        echo \
          "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
          $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
          sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
        sudo apt-get update
        sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
        ```
    *   Add your user to the `docker` group to run Docker commands without `sudo`:
        ```bash
        sudo usermod -aG docker your_user
        ```
    *   Log out and log back in again for the group changes to take effect.

## 2. Data Backup and Transfer

1.  **Stop the Application (Old Server):**
    *   To ensure data consistency, stop the entire stack on your current server.
        ```bash
        docker stack rm github-trending
        ```
2.  **Archive Data (Old Server):**
    *   Create a compressed archive of the `data` directory.
        ```bash
        tar -czvf github-trending-data.tar.gz data/
        ```
3.  **Transfer Data (From Old Server to New):**
    *   Use `scp` to securely copy the data archive to the new server.
        ```bash
        scp github-trending-data.tar.gz your_user@<new_server_ip>:~/
        ```

## 3. Application Setup on New Server

1.  **Transfer Project Files:**
    *   On the new server, clone your project from its Git repository.
        ```bash
        git clone <your_git_repository_url> github-trending
        cd github-trending
        ```
2.  **Restore Data:**
    *   Move the data archive into the project directory and extract it.
        ```bash
        mv ~/github-trending-data.tar.gz .
        tar -xzvf github-trending-data.tar.gz
        ```
3.  **Configure Environment:**
    *   Create the `.env` file. **Do not** copy this file from public sources. Recreate it with your secrets.
        ```bash
        cp .env.example .env
        nano .env
        ```
    *   Fill in all the required variables (passwords, tokens, etc.) just as they were on the old server.
4.  **Initialize Docker Swarm:**
    *   The application is deployed as a Swarm stack, so you must initialize Swarm on the new server.
        ```bash
        docker swarm init
        ```

## 4. Build and Deploy

1.  **Build ARM64 Images:**
    *   Because the new server has a different architecture (ARM64), you **must** rebuild all the Docker images on the new machine.
        ```bash
        docker compose build
        ```
2.  **Deploy the Stack:**
    *   Deploy the application using the `docker-compose.yml` file.
        ```bash
        docker stack deploy -c docker-compose.yml github-trending
        ```

## 5. Verification

1.  **Check Services:**
    *   Wait a few minutes for all services to start, then check their status.
        ```bash
        docker stack services github-trending
        ```
    *   Ensure all services show `1/1` replicas (or more for scaled services like `crawler`).
2.  **Inspect Logs:**
    *   Check the logs for a few key services to ensure there are no errors.
        ```bash
        docker service logs github-trending_api-server
        docker service logs github-trending_postgres_db
        ```
3.  **Test Application:**
    *   Access the `api-server` or the web UI to confirm the application is running correctly and that all previous data is available.
