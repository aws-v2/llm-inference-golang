#!/bin/sh

set -e

echo "Running database migrations..."

migrate -path ./internal/migrations \
  -database "postgres://postgres-staging-user:postgres-staging-password@localhost:5432/llm_db?sslmode=disable" up

echo "Starting application..."


echo "Starting application..."

./go-layer