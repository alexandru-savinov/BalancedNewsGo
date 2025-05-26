# Deployment Guide

This guide provides instructions for deploying the NewsBalancer system in production environments.

## Production Deployment Checklist

Before deploying to production, ensure you've completed these steps:

- [ ] Set up proper API keys with sufficient quota
- [ ] Configure a production-grade database setup
- [ ] Set all required environment variables
- [ ] Enable release mode for the server
- [ ] **Verify Editorial template assets are properly served**
- [ ] **Test responsive design across different devices**
- [ ] **Validate template rendering performance**
- [ ] Implement proper monitoring
- [ ] Set up automated backups
- [ ] Configure appropriate logging
- [ ] Implement rate limiting and security measures

## Environment Configuration

### Required Environment Variables

```bash
# Production mode - reduces verbose output, improves performance
export GIN_MODE=release

# LLM API configuration
export LLM_API_KEY=your_production_api_key
export LLM_API_KEY_SECONDARY=your_backup_api_key  # For API key rotation
export LLM_HTTP_TIMEOUT=30s  # Appropriate timeout for production

# Optional: Database path (default is "news.db" in working directory)
export DATABASE_URL=/path/to/production/database.db

# Web interface mode (Editorial templates are default, legacy client-side available)
# export LEGACY_HTML=true  # Only set if you need legacy client-side rendering mode
```

### Production Configuration Files

1. **feed_sources.json**: Review and update with production-worthy news sources
2. **composite_score_config.json**: Tune the ensemble configuration for production use
3. **Editorial Templates**: Ensure all template files in `web/templates/` are present and properly formatted
4. **Static Assets**: Verify that all assets in `web/assets/` (CSS, JS, images) are included in deployment

## Docker Deployment

### Dockerfile

Create a `Dockerfile` in the project root:

```dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o server ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/server /app/
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/web /app/web

# Make sure these directories exist
RUN mkdir -p /app/data

# Set environment variables
ENV GIN_MODE=release

EXPOSE 8080
VOLUME ["/app/data"]

CMD ["./server"]
```

### Docker Compose Configuration

Create `docker-compose.yml`:

```yaml
version: '3'

services:
  newsbalancer:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
      - LLM_API_KEY=${LLM_API_KEY}
      - LLM_API_KEY_SECONDARY=${LLM_API_KEY_SECONDARY}
      - LLM_HTTP_TIMEOUT=30s
      - DATABASE_URL=/app/data/news.db
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

### Building and Running with Docker

```bash
# Build the image
docker build -t newsbalancer .

# Run with Docker
docker run -p 8080:8080 \
  -e LLM_API_KEY=your_api_key \
  -e GIN_MODE=release \
  -v $(pwd)/data:/app/data \
  newsbalancer

# Or with Docker Compose
docker-compose up -d
```

## Database Configuration

### SQLite Production Setup

For production SQLite usage:

1. Use a dedicated volume or persistent storage path for the database
2. Implement regular backups:
   ```bash
   # Example backup script
   sqlite3 news.db ".backup '/backup/news-$(date +%Y%m%d).db'"
   ```
3. Consider database optimization:
   ```bash
   # Example optimization script
   sqlite3 news.db "PRAGMA optimize;"
   ```

### Database Migration

If migrating from development to production:

1. Export schema:
   ```bash
   sqlite3 dev.db .schema > schema.sql
   ```
2. Create new production database:
   ```bash
   sqlite3 production.db < schema.sql
   ```
3. Export and import data if necessary:
   ```bash
   # Export data
   sqlite3 dev.db ".mode insert" ".output data.sql" "SELECT * FROM articles;"

   # Import to production
   sqlite3 production.db < data.sql
   ```

## Performance Tuning

### Go Application Performance

1. **Memory Allocation**: Adjust Go memory settings if needed:
   ```bash
   # Increase max heap size
   export GOGC=100
   ```

2. **Connection Pooling**: The application already sets sane defaults, but monitor and adjust as needed:
   ```go
   // In internal/db/db.go
   db.SetMaxOpenConns(20)  // Increase from 10 for higher traffic
   db.SetMaxIdleConns(10)  // Increase from 5 for higher traffic
   ```

3. **LLM Service Timeout**: Adjust based on observed performance:
   ```bash
   export LLM_HTTP_TIMEOUT=45s  # Increase from default if needed
   ```

### Web Server Performance

1. **Gin Release Mode**:
   ```bash
   export GIN_MODE=release
   ```

2. **Proxy Integration**: Use with a reverse proxy (Nginx, Caddy) for SSL and better HTTP performance:

   **Nginx Example Config**:
   ```nginx
   server {
       listen 80;
       server_name newsbalancer.example.com;

       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

## Monitoring Setup

### Prometheus Integration

The system exposes Prometheus metrics. Configure a Prometheus instance to scrape:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'newsbalancer'
    scrape_interval: 15s
    static_configs:
      - targets: ['newsbalancer:8080']
```

### Custom Health Checks

The system exposes health endpoints:

- `/healthz` - Basic health check
- `/api/feeds/healthz` - RSS feed health status

Set up monitoring to regularly check these endpoints.

### Log Management

For production, redirect logs to a log management system:

```bash
# Redirect stdout/stderr to log files
./server > /var/log/newsbalancer/app.log 2> /var/log/newsbalancer/error.log

# Or use a log rotation tool
./server 2>&1 | rotatelogs /var/log/newsbalancer/app.%Y%m%d.log 86400
```

## Security Considerations

1. **API Key Protection**
   - Store keys securely (environment variables or secrets management)
   - Rotate keys periodically
   - Use least-privilege keys with appropriate quotas

2. **Rate Limiting**
   - Implement API rate limiting (can be added to the Gin router)
   - Example middleware:
     ```go
     func RateLimitMiddleware() gin.HandlerFunc {
         // Implement rate limiting logic here
     }
     ```

3. **Input Validation**
   - The system already validates inputs, but monitor for potential injection attacks
   - Ensure proper sanitization for all user inputs

## Scaling Strategies

1. **Vertical Scaling**
   - Increase resources (CPU, memory) for the server
   - Optimize SQLite with faster storage

2. **Horizontal Scaling Options**
   - Deploy multiple read-only instances with shared database
   - Consider moving to a distributed database if SQLite becomes a bottleneck
   - Implement a load balancer for API traffic distribution

3. **Caching Strategies**
   - The system already implements caching, but tune the TTL values based on traffic patterns
   - Consider adding a Redis cache for shared caching in a multi-instance setup

## Backup and Recovery

1. **Database Backup**
   ```bash
   # Regular backups (SQLite only)
   sqlite3 /app/data/news.db ".backup '/backup/news-$(date +%Y%m%d).db'"

   # Keep last 7 days of backups
   find /backup -name "news-*.db" -mtime +7 -delete
   ```

2. **Configuration Backup**
   - Version control your configuration files
   - Back up `.env` files securely

3. **Recovery Testing**
   - Regularly test database restoration procedures
   - Document and practice recovery steps
