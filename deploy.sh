#!/bin/bash

# Exit on error
set -e

echo "--- Building images ---"
docker compose build

echo "--- Deploying stack: github-trending ---"
# --prune removes services no longer defined in the compose file
# --with-registry-auth is useful if you use a private registry later
docker stack deploy -c docker-compose.yml github-trending

echo "--- Deployment status ---"
docker stack services github-trending
