#!/bin/bash

# Exit on error
set -e

echo "--- Building images ---"
docker compose build --pull

# docker stack deploy does not interpolate variables from .env on its own.
# Export the values used by docker-compose.yml without sourcing the whole file
# (values such as LAST_UPDATE_CUT contain spaces).
POSTGRES_USER="$(sed -n 's/^POSTGRES_USER=//p' .env)"
POSTGRES_DB="$(sed -n 's/^POSTGRES_DB=//p' .env)"
REDIS_PASSWORD="$(sed -n 's/^REDIS_PASSWORD=//p' .env)"
S3_ACCESS_KEY="$(sed -n 's/^S3_ACCESS_KEY=//p' .env)"
S3_SECRET_KEY="$(sed -n 's/^S3_SECRET_KEY=//p' .env)"
export POSTGRES_USER POSTGRES_DB REDIS_PASSWORD S3_ACCESS_KEY S3_SECRET_KEY

echo "--- Deploying stack: github-trending ---"
# --resolve-image never keeps locally built images local instead of trying to
#   record registry digests for them.
docker stack deploy --resolve-image never -c docker-compose.yml github-trending

echo "--- Rolling out rebuilt local images ---"
# With --resolve-image never the service spec (github-trending/<svc>:latest)
# does not change when an image is rebuilt, so Swarm keeps the old containers.
# Force-update every locally built service whose containers are not running
# the freshly built image.
for svc in $(docker stack services github-trending --format '{{.Name}}'); do
  image="$(docker service inspect "$svc" --format '{{.Spec.TaskTemplate.ContainerSpec.Image}}')"
  case "$image" in github-trending/*) ;; *) continue ;; esac
  want="$(docker image inspect "${image%@*}" --format '{{.Id}}' 2>/dev/null)" || continue
  for c in $(docker ps -q --filter "label=com.docker.swarm.service.name=$svc"); do
    if [ "$(docker inspect "$c" --format '{{.Image}}')" != "$want" ]; then
      echo "Updating $svc to the rebuilt image"
      docker service update --quiet --detach --force "$svc" >/dev/null
      break
    fi
  done
done

echo "--- Deployment status ---"
docker stack services github-trending
