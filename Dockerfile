# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -o go-layer ./cmd/gateway


# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wget tar bash

# Install migrate CLI tool
RUN wget -qO- https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz \
    | tar xvz && mv migrate /usr/local/bin/

# Copy binary and assets
COPY --from=builder /app/go-layer .
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/internal/migrations ./internal/migrations

# Copy entrypoint script
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8087

# Run migrations + start app
CMD ["./entrypoint.sh"]