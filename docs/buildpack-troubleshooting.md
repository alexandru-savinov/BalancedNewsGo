# Buildpack Troubleshooting Guide

This guide covers common issues and solutions when using Cloud Native Buildpacks with NewsBalancer.

## Common Build Issues

### 1. Pack CLI Not Found

**Error:**
```
'pack' is not recognized as an internal or external command
```

**Solution:**
```bash
# Windows - ensure pack.exe is in the correct location
./pack-cli/pack.exe version

# Linux/macOS - check installation
which pack
pack version

# If not installed, follow installation guide in docs/deployment.md
```

### 2. Builder Not Found

**Error:**
```
ERROR: failed to build: invalid builder 'paketobuildpacks/builder-jammy-base'
```

**Solution:**
```bash
# Set default builder
pack config default-builder paketobuildpacks/builder-jammy-base

# Or specify builder explicitly
pack build newsbalancer:test --builder paketobuildpacks/builder-jammy-base --path .
```

### 3. Go Version Compatibility

**Error:**
```
ERROR: failed to build: go version go1.xx.x is not supported
```

**Solution:**
Update `project.toml` with compatible Go version:
```toml
[[build.env]]
name = "BP_GO_VERSION"
value = "1.23.*"  # Use latest 1.23.x version
```

### 4. Build Target Issues

**Error:**
```
ERROR: failed to build: no Go files found in ./cmd/nonexistent
```

**Solution:**
Verify build targets in `project.toml`:
```toml
[[build.env]]
name = "BP_GO_TARGETS"
value = "./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"
```

Check that all target directories exist:
```bash
ls -la cmd/
```

### 5. File Preservation Issues

**Error:**
```
ERROR: template files not found in container
```

**Solution:**
Ensure `BP_KEEP_FILES` includes all necessary files:
```toml
[[build.env]]
name = "BP_KEEP_FILES"
value = "templates/*:static/*:configs/*:.env:*.db"
```

Verify files exist before build:
```bash
ls -la templates/ static/ configs/
```

## Common Runtime Issues

### 1. Container Exits Immediately

**Symptoms:**
```bash
docker run newsbalancer:test
# Container exits without output
```

**Diagnosis:**
```bash
# Check container logs
docker logs <container-id>

# Run with interactive mode for debugging
docker run -it --rm newsbalancer:test /bin/bash
# Note: This won't work with distroless images
```

**Common Causes:**
- Missing environment variables (LLM_API_KEY, DB_CONNECTION)
- Database file permissions
- Port conflicts

**Solution:**
```bash
# Run with required environment variables
docker run --rm \
  -e LLM_API_KEY="test-key" \
  -e DB_CONNECTION="/tmp/test.db" \
  -p 8080:8080 \
  newsbalancer:test
```

### 2. Database Connection Issues

**Error:**
```
ERROR: failed to open database: unable to open database file
```

**Solutions:**

**Check file permissions:**
```bash
# Ensure data directory is writable
mkdir -p /opt/newsbalancer/data
chmod 755 /opt/newsbalancer/data

# Run with proper volume mount
docker run -d \
  -v /opt/newsbalancer/data:/data \
  -e DB_CONNECTION="/data/newsbalancer.db" \
  newsbalancer:test
```

**Use in-memory database for testing:**
```bash
docker run --rm \
  -e DB_CONNECTION=":memory:" \
  newsbalancer:test
```

### 3. Template/Static File Issues

**Error:**
```
ERROR: template not found: templates/index.html
```

**Diagnosis:**
```bash
# Check if files are included in container
docker run --rm newsbalancer:test ls -la /workspace/templates/
docker run --rm newsbalancer:test ls -la /workspace/static/
```

**Solution:**
Verify `BP_KEEP_FILES` configuration and rebuild:
```bash
# Clean and rebuild
make buildpack-clean
make buildpack-build
```

### 4. Port Binding Issues

**Error:**
```
ERROR: bind: address already in use
```

**Solution:**
```bash
# Check what's using the port
netstat -tulpn | grep :8080

# Use different port
docker run -p 8081:8080 newsbalancer:test

# Or stop conflicting services
docker stop $(docker ps -q --filter "publish=8080")
```

### 5. API Key Validation Issues

**Error:**
```
ERROR: API key validation failed
```

**Solution:**
```bash
# For testing, use test mode
docker run --rm \
  -e TEST_MODE=true \
  -e LLM_API_KEY="test-key" \
  newsbalancer:test

# For production, ensure valid API key
docker run --rm \
  -e LLM_API_KEY="your-real-api-key" \
  newsbalancer:test
```

## Performance Issues

### 1. Slow Build Times

**Symptoms:**
- Build takes >5 minutes
- Frequent re-downloading of dependencies

**Solutions:**

**Enable Docker BuildKit:**
```bash
export DOCKER_BUILDKIT=1
```

**Use build cache:**
```bash
# Builds will be faster on subsequent runs due to layer caching
pack build newsbalancer:test --path .
```

**Optimize Go modules:**
```bash
# Clean module cache if corrupted
go clean -modcache
go mod download
```

### 2. Large Image Size

**Symptoms:**
- Image size >500MB
- Slow container startup

**Solutions:**

**Verify build optimization:**
```toml
# Ensure LDFLAGS are set for smaller binaries
[[build.env]]
name = "BP_GO_BUILD_LDFLAGS"
value = "-w -s"

# Disable CGO for static binaries
[[build.env]]
name = "CGO_ENABLED"
value = "0"
```

**Check image layers:**
```bash
docker images newsbalancer:test
docker history newsbalancer:test
```

### 3. Slow Runtime Performance

**Symptoms:**
- High response times
- High memory usage

**Solutions:**

**Monitor resource usage:**
```bash
docker stats newsbalancer-container
```

**Optimize database:**
```bash
# Ensure WAL mode is enabled (automatic in application)
# Consider external database for production
```

**Tune container resources:**
```bash
docker run --memory=1g --cpus=0.5 newsbalancer:test
```

## Debugging Techniques

### 1. Verbose Build Logging

```bash
# Enable verbose buildpack logging
pack build newsbalancer:debug --path . --verbose

# Check specific buildpack logs
pack build newsbalancer:debug --path . --log-level debug
```

### 2. Container Inspection

```bash
# Inspect running container
docker exec -it newsbalancer-container ps aux
docker exec -it newsbalancer-container env

# Check file system
docker exec -it newsbalancer-container find /workspace -type f -name "*.html"
```

### 3. Network Debugging

```bash
# Test connectivity from inside container
docker exec -it newsbalancer-container curl -I http://localhost:8080/healthz

# Check port binding
docker port newsbalancer-container
```

### 4. Application Logs

```bash
# Follow logs in real-time
docker logs -f newsbalancer-container

# Get recent logs
docker logs --tail 50 newsbalancer-container

# Filter logs
docker logs newsbalancer-container 2>&1 | grep ERROR
```

## Makefile Troubleshooting

### 1. Make Command Not Found

**Windows:**
```powershell
# Install make for Windows
choco install make
# Or use nmake (Visual Studio)
# Or run commands directly
```

### 2. Makefile Target Issues

```bash
# List available targets
make help

# Run with verbose output
make buildpack-build VERBOSE=1

# Check Makefile syntax
make -n buildpack-build
```

## CI/CD Issues

### 1. GitHub Actions Build Failures

**Check workflow logs:**
```bash
gh run list --workflow="CI/CD Pipeline"
gh run view <run-id> --log
```

**Common fixes:**
- Ensure Pack CLI is installed in CI
- Verify builder is available
- Check environment variables are set

### 2. Registry Push Issues

```bash
# Login to registry
docker login your-registry.com

# Tag image properly
docker tag newsbalancer:latest your-registry.com/newsbalancer:v1.0.0

# Push with explicit tag
docker push your-registry.com/newsbalancer:v1.0.0
```

## Getting Help

### 1. Collect Debug Information

```bash
# System information
pack version
docker version
go version

# Build information
pack inspect newsbalancer:test

# Runtime information
docker inspect newsbalancer-container
```

### 2. Enable Debug Mode

```bash
# Build with debug information
pack build newsbalancer:debug --path . --env BP_LOG_LEVEL=DEBUG

# Run with debug logging
docker run --rm \
  -e DEBUG=true \
  -e LOG_LEVEL=debug \
  newsbalancer:debug
```

### 3. Common Log Locations

- **Build logs**: Pack CLI output
- **Runtime logs**: Docker logs or container stdout/stderr
- **Application logs**: Application-specific logging to stdout

### 4. Useful Commands for Support

```bash
# Generate support bundle
{
  echo "=== System Info ==="
  pack version
  docker version
  go version
  
  echo "=== Build Info ==="
  pack inspect newsbalancer:test 2>/dev/null || echo "Image not found"
  
  echo "=== Runtime Info ==="
  docker logs newsbalancer-container 2>/dev/null || echo "Container not running"
} > support-bundle.txt
```

For additional help, see:
- [Main README](../README.md)
- [Deployment Guide](deployment.md)
- [Testing Guide](testing.md)
- [Pack CLI Documentation](https://buildpacks.io/docs/tools/pack/)
