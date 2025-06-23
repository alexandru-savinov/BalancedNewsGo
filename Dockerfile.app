# Multi-stage Dockerfile for NewsBalancer Go Application
# This Dockerfile creates optimized production images with minimal size

# Stage 1: Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o newsbalancer \
    ./cmd/server

# Build benchmark tool
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o benchmark \
    ./cmd/benchmark

# Stage 2: Production stage
FROM scratch AS production

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from builder stage
COPY --from=builder /app/newsbalancer /newsbalancer

# Copy configuration files directly from build context
COPY configs /configs
COPY templates /templates
COPY static /static

# Debug: List what we have
RUN ls -la / && ls -la /configs && ls -la /templates && ls -la /static

# Create a non-root user (using numeric IDs for scratch image)
USER 65534:65534

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/newsbalancer", "--health-check"]

# Run the application
ENTRYPOINT ["/newsbalancer"]

# Stage 3: Development stage (includes debugging tools)
FROM golang:1.23-alpine AS development

# Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    bash \
    make \
    gcc \
    musl-dev

# Install Go tools
RUN go install github.com/air-verse/air@latest
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Use air for hot reloading in development
CMD ["air", "-c", ".air.toml"]

# Stage 4: Testing stage (includes test dependencies)
FROM golang:1.23-alpine AS testing

# Install test dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    bash \
    make \
    gcc \
    musl-dev \
    postgresql-client

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Run tests
RUN go test -v ./...

# Stage 5: Benchmark stage (includes benchmark tools)
FROM production AS benchmark

# Copy benchmark binary
COPY --from=builder /app/benchmark /benchmark

# Copy benchmark configuration
COPY --from=builder /app/benchmark-config.json /benchmark-config.json

# Default command runs benchmarks
CMD ["/benchmark", "-config", "/benchmark-config.json"]

# Stage 6: Debug stage (includes debugging tools but smaller than development)
FROM alpine:3.18 AS debug

# Install minimal debugging tools
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    bash

# Copy the binary from builder stage
COPY --from=builder /app/newsbalancer /newsbalancer

# Copy configuration files directly from build context
COPY configs /configs
COPY templates /templates
COPY static /static

# Debug: List what we have
RUN ls -la / && ls -la /configs && ls -la /templates && ls -la /static

# Create a non-root user
RUN addgroup -g 1001 -S newsbalancer && \
    adduser -u 1001 -S newsbalancer -G newsbalancer

USER newsbalancer:newsbalancer

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/articles || exit 1

# Run the application
ENTRYPOINT ["/newsbalancer"]
