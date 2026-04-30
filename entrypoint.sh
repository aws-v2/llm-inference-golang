#!/bin/sh

set -e

echo "Running database migrats"

migrate -path ./internal/migrations \
  -database "postgres://postgres-prod-user:postgres-prod-password@postgres-prod:5432/llm_db?sslmode=disable" up

echo "Starting application..."


echo "Starting application..."

./go-layer