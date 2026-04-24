#!/bin/sh

set -e

echo "Running database migrations..."

migrate -path ./internal/migrations \
  -database "$DB_URL" up

echo "Starting application..."

./go-layer