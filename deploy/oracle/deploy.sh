#!/usr/bin/env bash
# Zero-downtime deploy. Called by GitHub Actions after image push.
set -euo pipefail

IMAGE="ghcr.io/hopeisco0l/anu-pro:latest"
ENV_FILE="/opt/anu/.env"

echo "==> Pulling $IMAGE"
docker pull "$IMAGE"

echo "==> Running migrations"
docker run --rm --env-file "$ENV_FILE" "$IMAGE" /app/migrate up public

echo "==> Restarting services"
systemctl restart anu-api anu-worker

echo "==> Health check"
sleep 5
for i in 1 2 3 4 5; do
    if curl -sf http://localhost:8080/health; then
        echo "==> Deploy successful"
        exit 0
    fi
    sleep 2
done

echo "ERROR: health check failed after deploy" >&2
exit 1
