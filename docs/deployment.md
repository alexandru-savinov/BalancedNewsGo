# NewsBalancer Deployment Guide

This guide covers deployment of NewsBalancer using Cloud Native Buildpacks, the recommended approach for production deployments.

## Overview

NewsBalancer has migrated from Docker to **Cloud Native Buildpacks** for improved:
- **Security**: Distroless runtime with minimal attack surface
- **Performance**: Optimized builds with layer caching
- **Maintainability**: Automatic dependency management and updates
- **Standardization**: Industry-standard buildpack approach

## Prerequisites

### System Requirements
- **Docker**: Version 20.10+ with BuildKit support
- **Pack CLI**: Version 0.38.2+ (installation instructions below)
- **Memory**: Minimum 2GB RAM for build process
- **Storage**: 5GB+ free space for images and layers

### Pack CLI Installation

**Windows (PowerShell):**
```powershell
# Download and install Pack CLI
$packVersion = "v0.38.2"
$packUrl = "https://github.com/buildpacks/pack/releases/download/$packVersion/pack-$packVersion-windows.zip"
Invoke-WebRequest -Uri $packUrl -OutFile "pack.zip"
Expand-Archive -Path "pack.zip" -DestinationPath "./pack-cli"
Remove-Item "pack.zip"

# Configure default builder
./pack-cli/pack.exe config default-builder paketobuildpacks/builder-jammy-base
./pack-cli/pack.exe version
```

**Linux:**
```bash
# Install Pack CLI
curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.38.2/pack-v0.38.2-linux.tgz" | sudo tar -C /usr/local/bin/ --no-same-owner -xzv pack

# Configure default builder
pack config default-builder paketobuildpacks/builder-jammy-base
pack version
```

**macOS:**
```bash
# Using Homebrew (recommended)
brew install buildpacks/tap/pack

# Or manual installation
curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.38.2/pack-v0.38.2-macos.tgz" | sudo tar -C /usr/local/bin/ --no-same-owner -xzv pack

# Configure default builder
pack config default-builder paketobuildpacks/builder-jammy-base
pack version
```

## Build Process

### Local Development Build

```bash
# Quick build and test
make buildpack-build
make buildpack-test

# Or build and run for development
make buildpack-run
```

### Production Build

```bash
# Build production-ready image
pack build newsbalancer:latest --path .

# Tag for registry
docker tag newsbalancer:latest your-registry.com/newsbalancer:v1.0.0

# Push to registry
docker push your-registry.com/newsbalancer:v1.0.0
```

### Build Configuration

The build process is configured via `project.toml`:

```toml
schema-version = "0.2"

[project]
id = "balanced-news-go"
name = "BalancedNewsGo"
version = "1.0.0"

# Buildpack configuration
[[build.buildpacks]]
uri = "docker://paketobuildpacks/go"

# Build environment
[[build.env]]
name = "BP_GO_VERSION"
value = "1.23.*"

[[build.env]]
name = "BP_GO_TARGETS"
value = "./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"

[[build.env]]
name = "BP_GO_BUILD_LDFLAGS"
value = "-w -s"

[[build.env]]
name = "BP_KEEP_FILES"
value = "templates/*:static/*:configs/*:.env:*.db"

[[build.env]]
name = "CGO_ENABLED"
value = "0"

[[build.env]]
name = "PORT"
value = "8080"
```

## Deployment Platforms

### Docker/Podman

**Basic Deployment:**
```bash
# Run with basic configuration
docker run -d --name newsbalancer \
  -p 8080:8080 \
  -e LLM_API_KEY="your-api-key" \
  -e DB_CONNECTION="/data/newsbalancer.db" \
  -v /host/data:/data \
  newsbalancer:latest
```

**Production Deployment with Monitoring:**
```bash
# Create data directory
mkdir -p /opt/newsbalancer/data

# Run with full configuration
docker run -d --name newsbalancer-prod \
  --restart unless-stopped \
  -p 8080:8080 \
  -e PORT=8080 \
  -e DB_CONNECTION="/data/newsbalancer.db" \
  -e LLM_API_KEY="your-api-key" \
  -e LLM_API_KEY_SECONDARY="your-secondary-key" \
  -e LLM_BASE_URL="https://your-llm-service.com" \
  -v /opt/newsbalancer/data:/data \
  --log-driver json-file \
  --log-opt max-size=100m \
  --log-opt max-file=3 \
  newsbalancer:latest
```

### Kubernetes

**Deployment Manifest:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: newsbalancer-config
data:
  PORT: "8080"
  DB_CONNECTION: "/data/newsbalancer.db"
---
apiVersion: v1
kind: Secret
metadata:
  name: newsbalancer-secrets
type: Opaque
stringData:
  llm-api-key: "your-api-key"
  llm-api-key-secondary: "your-secondary-key"
  llm-base-url: "https://your-llm-service.com"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: newsbalancer-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: newsbalancer
  labels:
    app: newsbalancer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: newsbalancer
  template:
    metadata:
      labels:
        app: newsbalancer
    spec:
      containers:
      - name: newsbalancer
        image: newsbalancer:latest
        ports:
        - containerPort: 8080
          name: http
        envFrom:
        - configMapRef:
            name: newsbalancer-config
        env:
        - name: LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: newsbalancer-secrets
              key: llm-api-key
        - name: LLM_API_KEY_SECONDARY
          valueFrom:
            secretKeyRef:
              name: newsbalancer-secrets
              key: llm-api-key-secondary
        - name: LLM_BASE_URL
          valueFrom:
            secretKeyRef:
              name: newsbalancer-secrets
              key: llm-base-url
        volumeMounts:
        - name: data-volume
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "500m"
      volumes:
      - name: data-volume
        persistentVolumeClaim:
          claimName: newsbalancer-data
---
apiVersion: v1
kind: Service
metadata:
  name: newsbalancer-service
spec:
  selector:
    app: newsbalancer
  ports:
  - port: 80
    targetPort: 8080
    name: http
  type: ClusterIP
```

**Ingress (Optional):**
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: newsbalancer-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: newsbalancer.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: newsbalancer-service
            port:
              number: 80
```

## Environment Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `LLM_API_KEY` | Primary LLM service API key | `sk-...` |
| `DB_CONNECTION` | Database file path | `/data/newsbalancer.db` |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `LLM_API_KEY_SECONDARY` | Secondary LLM API key | - |
| `LLM_BASE_URL` | Custom LLM service URL | - |
| `NO_AUTO_ANALYZE` | Disable automatic analysis | `false` |

### Configuration Files

The application expects these files to be available in the container:
- `/workspace/configs/feed_sources.json` - RSS feed configuration
- `/workspace/configs/composite_score_config.json` - LLM scoring configuration
- `/workspace/templates/` - HTML templates
- `/workspace/static/` - Static assets (CSS, JS, images)

These files are automatically included via the `BP_KEEP_FILES` buildpack configuration.

## Monitoring and Observability

### Health Checks

The application provides a health check endpoint:
```bash
curl http://localhost:8080/healthz
```

### Logging

Application logs are written to stdout/stderr and can be collected by your container platform:

```bash
# Docker logs
docker logs newsbalancer-prod

# Kubernetes logs
kubectl logs deployment/newsbalancer
```

### Metrics

Key metrics endpoints:
- `/api/articles` - Article count and status
- `/api/feeds/healthz` - RSS feed health status

### Optional Monitoring Stack

A complete monitoring stack is available in `monitoring/docker-compose.monitoring.yml`:

```bash
# Start monitoring stack
docker compose -f monitoring/docker-compose.monitoring.yml up -d

# Access Grafana
open http://localhost:3000
```

## Troubleshooting

### Common Issues

**1. Build Failures:**
```bash
# Check buildpack logs
pack build newsbalancer:debug --path . --verbose

# Verify Go version compatibility
go version
```

**2. Runtime Issues:**
```bash
# Check container logs
docker logs newsbalancer-prod

# Verify environment variables
docker exec newsbalancer-prod env | grep -E "(LLM|DB|PORT)"

# Test health endpoint
curl -f http://localhost:8080/healthz
```

**3. Database Issues:**
```bash
# Check database file permissions
docker exec newsbalancer-prod ls -la /data/

# Verify database connectivity
docker exec newsbalancer-prod sqlite3 /data/newsbalancer.db ".tables"
```

**4. Performance Issues:**
```bash
# Check resource usage
docker stats newsbalancer-prod

# Monitor application metrics
curl http://localhost:8080/api/articles | jq '.total'
```

### Debug Mode

For debugging, you can run the container with additional logging:

```bash
docker run -it --rm \
  -e DEBUG=true \
  -e LOG_LEVEL=debug \
  newsbalancer:latest
```

## Security Considerations

### Container Security
- **Distroless Runtime**: Minimal attack surface with no shell access
- **Non-root User**: Application runs as non-privileged user
- **Read-only Filesystem**: Most filesystem areas are read-only

### Secrets Management
- Use container orchestration secrets (Kubernetes secrets, Docker secrets)
- Never embed API keys in images
- Rotate API keys regularly

### Network Security
- Use TLS termination at load balancer/ingress
- Implement network policies in Kubernetes
- Restrict container-to-container communication

## Performance Tuning

### Build Optimization
- **Layer Caching**: Buildpacks automatically optimize layer caching
- **Multi-stage Builds**: Not needed with buildpacks (handled automatically)
- **Binary Optimization**: LDFLAGS configured for minimal binary size

### Runtime Optimization
- **Memory**: Start with 256MB, scale based on usage
- **CPU**: 100m request, 500m limit for most workloads
- **Database**: Use WAL mode for better concurrency (automatically configured)

### Scaling Considerations
- **Horizontal Scaling**: Application is stateless and scales horizontally
- **Database**: Consider external database for multi-instance deployments
- **Load Balancing**: Use sticky sessions if needed for WebSocket connections

For additional support, see the main [README.md](../README.md) and [troubleshooting documentation](troubleshooting/).
