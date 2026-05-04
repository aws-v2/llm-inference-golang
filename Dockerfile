# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o go-layer ./cmd/gateway




# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates bash

# Copy binary and assets
COPY --from=builder /app/go-layer .
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/internal/migrations ./internal/migrations

EXPOSE 8087

# Migrations are run by the Go binary at startup
CMD ["./go-layer"]