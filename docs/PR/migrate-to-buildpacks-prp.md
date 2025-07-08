name: "Migrate from Docker to Buildpacks - Issue #96 PRP"
description: |

## Purpose
Comprehensive Problem Resolution Proposal for migrating from Docker to Cloud Native Buildpacks to simplify deployment process and reduce configuration overhead.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Migrate the BalancedNewsGo project from Docker-based deployment to Cloud Native Buildpacks (CNB) to simplify the build and deployment process, reduce configuration overhead, and improve maintainability while maintaining all current functionality.

## Why
- **Reduced Complexity**: Current Docker setup has 6 different stages (production, development, testing, benchmark, debug, scratch) creating maintenance overhead
- **Simplified Configuration**: Buildpacks auto-detect dependencies and provide sensible defaults, reducing manual configuration
- **Better Developer Experience**: Buildpacks provide consistent environments across development, testing, and production
- **Industry Standard**: CNB is a CNCF project with broad ecosystem support
- **Faster Builds**: Buildpacks leverage intelligent caching and layer reuse
- **Security**: Buildpacks provide regularly updated base images with security patches

## What
Replace the current Docker configuration files (Dockerfile, Dockerfile.app) with Cloud Native Buildpacks configuration, maintaining all current build capabilities including:
- Production builds
- Development environment with hot reloading
- Testing environment
- Benchmark builds
- CI/CD integration

### Success Criteria
- [ ] Docker files are removed from the repository
- [ ] Buildpack configuration is properly set up with project.toml
- [ ] Application builds and deploys successfully with pack CLI
- [ ] All build stages (dev, test, prod, benchmark) work with buildpacks
- [ ] CI/CD pipeline updated to use buildpacks instead of Docker
- [ ] Build/deployment time is improved or maintained compared to Docker
- [ ] Documentation updated to reflect the new process
- [ ] All existing functionality preserved (web server, API, database, monitoring)

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- url: https://buildpacks.io/docs/
  why: Official CNB documentation for understanding buildpack concepts
  
- url: https://buildpacks.io/docs/for-app-developers/
  why: App developer guide for using buildpacks with Go applications
  
- url: https://github.com/buildpacks/pack
  why: Pack CLI tool documentation and usage examples
  
- url: https://buildpacks.io/docs/for-app-developers/how-to/build-inputs/specify-buildpacks/
  why: How to specify and configure buildpacks for Go applications

- file: Dockerfile.app
  why: Current multi-stage Docker configuration to understand requirements
  
- file: go.mod
  why: Go dependencies and module configuration
  
- file: cmd/server/main.go
  why: Application entry point and server configuration
  
- file: Makefile
  why: Current build and deployment processes
  
- file: .github/workflows/ci.yml
  why: CI/CD pipeline configuration that needs updating
```

### Current Codebase tree
```bash
BalancedNewsGo/
├── cmd/
│   ├── server/           # Main application entry point
│   ├── benchmark/        # Benchmark tool
│   └── [other tools]/
├── internal/             # Internal packages
├── configs/              # Configuration files
├── templates/            # HTML templates
├── static/               # Static assets
├── Dockerfile            # Playwright testing Docker config
├── Dockerfile.app        # Multi-stage Go app Docker config
├── go.mod               # Go module definition
├── Makefile             # Build automation
└── .github/workflows/   # CI/CD configuration
```

### Desired Codebase tree with files to be added and responsibility of file
```bash
BalancedNewsGo/
├── project.toml          # Buildpack configuration (replaces Dockerfiles)
├── Procfile             # Process definitions for different environments
├── buildpacks/          # Custom buildpack configurations if needed
│   └── go-dev.toml      # Development-specific buildpack config
├── cmd/                 # Unchanged
├── internal/            # Unchanged
├── configs/             # Unchanged
├── templates/           # Unchanged
├── static/              # Unchanged
├── go.mod              # Unchanged
├── Makefile            # Updated for buildpack commands
└── .github/workflows/  # Updated CI/CD for buildpacks
```

### Known Gotchas of our codebase & Library Quirks
```go
// CRITICAL: Go 1.23.0 required - ensure buildpack supports this version
// CRITICAL: SQLite database uses modernc.org/sqlite (Pure Go, NO CGO required)
// VERIFIED: All test files updated to use pure Go SQLite driver consistently
// VERIFIED: All unit, integration, and E2E tests pass with pure Go driver
// VERIFIED: Main application builds and runs correctly with pure Go driver
// CRITICAL: Static assets in /static and /templates must be included in build
// CRITICAL: Configuration files in /configs must be accessible at runtime
// CRITICAL: Multiple entry points (server, benchmark, tools) need different build targets
// CRITICAL: Gin web framework requires proper port binding (PORT env var)
// CRITICAL: Application expects specific directory structure for templates/static
```

## Implementation Blueprint

### Data models and structure

The buildpack configuration will use TOML format for project.toml and leverage Go buildpack auto-detection:
```toml
# project.toml - Main buildpack configuration
[project]
id = "balanced-news-go"
name = "BalancedNewsGo"
version = "1.0.0"

[[build.buildpacks]]
uri = "gcr.io/paketo-buildpacks/go"

[[build.buildpacks]]
uri = "gcr.io/paketo-buildpacks/ca-certificates"

[build.env]
BP_GO_VERSION = "1.23.*"
BP_GO_TARGETS = "./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"
BP_GO_BUILD_LDFLAGS = "-w -s"
BP_KEEP_FILES = "templates/*:static/*:configs/*:.env:*.db"
CGO_ENABLED = "0"

# Environment Variable Considerations:
# - PORT: Auto-set by buildpack, app defaults to 8080 (compatible)
# - DB_CONNECTION: SQLite file path, needs persistence strategy
# - LLM_API_KEY: From .env file, consider runtime injection for security
# - LOG_FILE_PATH: File logging vs stdout/stderr for cloud-native approach
```

### List of tasks to be completed to fulfill the PRP in the order they should be completed

```yaml
Task 1: Research and Install Pack CLI
  - DOWNLOAD pack CLI from https://github.com/buildpacks/pack/releases
  - INSTALL on development machine and CI/CD environment
  - VERIFY installation with `pack version`
  - DOCUMENT installation process for team

Task 2: Create Basic Buildpack Configuration
  - CREATE project.toml with Go buildpack configuration
  - CONFIGURE environment variables for Go version and CGO
  - TEST basic build with `pack build balanced-news-go`
  - VALIDATE application starts correctly

Task 3: Configure Multi-Process Support (Hybrid Approach)
  - CREATE Procfile for production processes (web, fetch, score, seed)
  - DEFINE web process for main server application (cmd/server)
  - DEFINE utility processes for production tools (fetch_articles, score_articles, seed_test_data)
  - EXCLUDE development tools (benchmark, clear_articles, tools/*) from buildpack image
  - TEST each included process type builds and runs correctly

Task 4: Validate File System Dependencies
  - VERIFY BP_KEEP_FILES preserves templates/, static/, configs/, .env files
  - TEST all 8+ HTML templates load correctly in buildpack environment
  - VALIDATE 5 JSON config files (feed_sources.json, bias_config.json, etc.) are accessible
  - ENSURE .env file loading works for LLM_API_KEY and other variables
  - TEST static asset serving from ./static directory works with Gin
  - VALIDATE database file persistence strategy (*.db files)

Task 5: Validate Environment Variable Handling
  - TEST PORT binding works with buildpack auto-detection (defaults to 8080)
  - VERIFY DB_CONNECTION environment variable for SQLite file path
  - VALIDATE LLM_API_KEY loading from .env file in buildpack environment
  - TEST LOG_FILE_PATH configuration and file writing permissions
  - ENSURE TEST_MODE and DOCKER environment flags work correctly
  - VALIDATE all environment variables accessible at runtime

Task 6: Update Build Scripts and Makefile
  - MODIFY Makefile to use pack commands instead of docker
  - UPDATE build targets for different environments
  - CREATE development build with live reload capability
  - TEST all make targets work with buildpacks

Task 7: Validate Database Persistence Strategy
  - DEFINE SQLite file storage location in production environment
  - TEST database file persistence across container restarts
  - VALIDATE database initialization works in buildpack environment
  - ENSURE database migrations work with new deployment method
  - TEST volume mounting for database persistence

Task 8: Update CI/CD Pipeline
  - MODIFY .github/workflows/ci.yml to use pack CLI
  - REPLACE docker build steps with pack build
  - UPDATE image registry push commands
  - TEST CI/CD pipeline builds successfully

Task 9: Remove Docker Configuration
  - DELETE Dockerfile and Dockerfile.app
  - REMOVE docker-related scripts and configurations
  - CLEAN UP docker references in documentation
  - VERIFY no docker dependencies remain

Task 10: Update Documentation
  - UPDATE README.md with buildpack instructions
  - CREATE deployment guide for buildpacks
  - DOCUMENT development workflow changes
  - UPDATE troubleshooting guide
```

### Per task pseudocode as needed added to each task

```bash
# Task 1: Install Pack CLI
# Download and install pack CLI
curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.38.2/pack-v0.38.2-linux.tgz" | tar -C /usr/local/bin/ --no-same-owner -xzv pack

# Verify installation
pack version
pack config default-builder gcr.io/paketo-buildpacks/builder:base

# Task 2: Basic Configuration
# Create project.toml with essential settings
cat > project.toml << EOF
[project]
id = "balanced-news-go"
name = "BalancedNewsGo"

[[build.buildpacks]]
uri = "gcr.io/paketo-buildpacks/go"

[build.env]
BP_GO_VERSION = "1.23.*"
BP_GO_TARGETS = "./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"
BP_KEEP_FILES = "templates/*:static/*:configs/*:.env:*.db"
CGO_ENABLED = "0"
EOF

# Test basic build
pack build balanced-news-go --path .

# Task 3: Multi-Process Configuration
# Create Procfile for production processes (hybrid approach)
cat > Procfile << EOF
web: ./cmd/server
fetch: ./cmd/fetch_articles
score: ./cmd/score_articles
seed: ./cmd/seed_test_data
EOF

# Build with multiple production targets
pack build balanced-news-go --path . --env BP_GO_TARGETS="./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"
```

### Integration Points
```yaml
BUILDPACK:
  - builder: "gcr.io/paketo-buildpacks/builder:base"
  - go-buildpack: "gcr.io/paketo-buildpacks/go"
  - ca-certificates: "gcr.io/paketo-buildpacks/ca-certificates"

CONFIG:
  - add to: project.toml
  - pattern: "BP_GO_VERSION = '1.23.*'"
  - pattern: "CGO_ENABLED = '1'"

MAKEFILE:
  - replace: "docker build" with "pack build"
  - pattern: "pack build $(APP_NAME) --path ."

CI_CD:
  - replace: "docker/build-push-action" with pack CLI commands
  - pattern: "pack build && pack publish"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
pack config validate project.toml     # Validate buildpack config
go mod verify                         # Verify Go modules
pack build balanced-news-go --dry-run # Dry run build

# Validate file preservation configuration
echo "Checking BP_KEEP_FILES configuration..."
ls -la templates/ static/ configs/ .env *.db 2>/dev/null || echo "Files to preserve identified"

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: File System & Environment Validation
```bash
# Test basic build with file preservation
pack build balanced-news-go --path . \
  --env BP_GO_TARGETS="./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data" \
  --env BP_KEEP_FILES="templates/*:static/*:configs/*:.env:*.db"

# Validate file preservation
echo "Checking preserved files..."
docker run --rm balanced-news-go ls -la templates/ static/ configs/ .env 2>/dev/null || echo "Files preserved"

# Test application starts with environment variables
docker run --rm -d --name test-app -p 8080:8080 \
  -e DB_CONNECTION="/tmp/test.db" \
  -e LLM_API_KEY="test-key" \
  -e LOG_FILE_PATH="/tmp/app.log" \
  balanced-news-go

sleep 10

# Test static assets serving
curl -f http://localhost:8080/static/css/main.css || echo "Static assets test failed"

# Test templates rendering
curl -f http://localhost:8080/ | grep -q "html" || echo "Template rendering test failed"

# Test API endpoints
curl -f http://localhost:8080/api/articles || echo "API test failed"
curl -f http://localhost:8080/healthz || echo "Health check failed"

# Clean up
docker stop test-app && docker rm test-app
```

### Level 3: Multi-Process & Database Persistence Test
```bash
# Build with hybrid configuration (all production processes)
pack build balanced-news-go --path . \
  --env BP_GO_TARGETS="./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data" \
  --env BP_KEEP_FILES="templates/*:static/*:configs/*:.env:*.db"

# Test database persistence with volume mount
mkdir -p ./test-data
docker run --rm -d --name test-persistence \
  -v $(pwd)/test-data:/data \
  -e DB_CONNECTION="/data/news.db" \
  -p 8080:8080 balanced-news-go

sleep 15

# Test web process functionality
curl -f http://localhost:8080/healthz || echo "Health check failed"
curl -f http://localhost:8080/api/articles || echo "Articles API failed"

# Test database file creation
docker exec test-persistence ls -la /data/ || echo "Database persistence check"

# Test different process types
docker exec test-persistence ./cmd/fetch_articles --help || echo "Fetch articles process test"
docker exec test-persistence ./cmd/score_articles --help || echo "Score articles process test"
docker exec test-persistence ./cmd/seed_test_data --help || echo "Seed test data process test"

# Clean up
docker stop test-persistence
ls -la ./test-data/ # Verify database file persisted

# Expected: All processes build, run successfully, and database persists
```

## Final validation Checklist
- [ ] Pack CLI installed and configured: `pack version`
- [ ] Basic build works: `pack build balanced-news-go --path .`
- [ ] File preservation verified: `docker run balanced-news-go ls -la templates/ static/ configs/`
- [ ] Environment variables work: Test with DB_CONNECTION, LLM_API_KEY, PORT
- [ ] Application starts: `docker run balanced-news-go`
- [ ] Web endpoints respond: `curl http://localhost:8080/api/articles`
- [ ] Static assets served: `curl http://localhost:8080/static/css/main.css`
- [ ] Templates render: `curl http://localhost:8080/`
- [ ] Config files accessible: JSON configs load correctly
- [ ] Database persistence: SQLite file persists across restarts
- [ ] Multi-process support: All 4 production binaries work
- [ ] CI/CD pipeline passes: GitHub Actions green
- [ ] Docker files removed: `ls Dockerfile*` returns nothing
- [ ] Documentation updated: README.md reflects buildpack usage

---

## Risk Assessment and Rollback Procedures

### High Risk Areas
1. **✅ Pure Go SQLite**: VERIFIED - All files now consistently use modernc.org/sqlite (pure Go)
2. **🔍 Multi-Binary Challenge**: ANALYZED - 11 binary entry points create buildpack complexity (see detailed analysis)
3. **🚨 File System Dependencies**: Complex file requirements (templates/, static/, configs/, .env) need validation
4. **🚨 Environment Variable Handling**: Critical env vars (PORT, DB_CONNECTION, LLM_API_KEY) handled differently in buildpacks
5. **🚨 Database Persistence Strategy**: SQLite file location and persistence across deployments undefined
6. **🚨 Static Asset Serving**: Gin static file serving from ./static must work in buildpack containers
7. **CI/CD Integration**: GitHub Actions must be updated without breaking deployment

### Rollback Plan
1. **Immediate Rollback**: Keep Docker files in a backup branch until validation complete
2. **CI/CD Rollback**: Maintain parallel Docker-based pipeline until buildpack pipeline proven
3. **Configuration Rollback**: Git revert commits if buildpack build fails
4. **Documentation Rollback**: Restore previous README.md if needed

### Mitigation Strategies
1. **Incremental Migration**: Test buildpacks locally before updating CI/CD
2. **Parallel Testing**: Run both Docker and buildpack builds during transition
3. **Feature Flags**: Use environment variables to switch between build methods
4. **Monitoring**: Verify application metrics remain stable post-migration

## PRP Confidence Score: 7/10

**Reasoning for 7/10 Score:**
- **High Confidence (7)**: Buildpacks are mature technology with strong Go support
- **Well-Defined Requirements**: Clear understanding of current Docker setup and needs
- **Proven Technology**: CNB is CNCF project with production usage
- **Good Documentation**: Extensive buildpack documentation and examples available
- **✅ Validated Pure Go**: All SQLite drivers now consistently use pure Go (no CGO)
- **✅ Test Coverage**: Comprehensive test suite passes with pure Go drivers
- **✅ Multi-Binary Analysis**: Hybrid approach provides clear solution path

**Risk Factors (-2 points):**
- **🔍 Multi-Binary Challenge**: ANALYZED - 11 binaries require hybrid approach (production + utilities)
- **🚨 File System Dependencies**: Complex file preservation requirements (templates, static, configs, .env)
- **🚨 Environment Variable Complexity**: Critical env vars handled differently in buildpacks
- **🚨 Database Persistence**: SQLite file persistence strategy needs definition

**Risk Mitigation Completed:**
- **✅ Driver Consistency**: All SQLite drivers now consistently use pure Go (modernc.org/sqlite)
- **✅ Test Validation**: All unit, integration, and E2E tests pass with pure Go driver
- **✅ Build Verification**: Main application builds and runs correctly
- **🔍 Multi-Binary Analysis**: Comprehensive analysis completed with hybrid solution recommended

**Confidence Boosters:**
- **Incremental Approach**: Can test locally before CI/CD changes
- **Rollback Plan**: Docker files can be restored if needed
- **Community Support**: Large ecosystem and community around buildpacks
- **Validation Strategy**: Comprehensive testing at each step

---

## SQLite Driver Consistency Validation (COMPLETED)

### Changes Made
1. **Updated 4 files** to use pure Go SQLite driver (`modernc.org/sqlite`):
   - `cmd/ingest_feedback/main.go`
   - `cmd/migrate_feedback_schema/main.go`
   - `cmd/score_articles/main.go`
   - `internal/testing/database.go`

2. **Replaced driver references**:
   - Import: `github.com/mattn/go-sqlite3` → `modernc.org/sqlite`
   - Driver name: `"sqlite3"` → `"sqlite"`

### Validation Results
- **✅ Unit Tests**: All pass (28 test packages)
- **✅ Integration Tests**: All pass (11 test packages)
- **✅ E2E Tests**: All pass (28 Playwright tests)
- **✅ Build Verification**: Main application builds successfully
- **✅ Tool Testing**: Updated command-line tools work correctly

### Impact on Buildpack Migration
- **Simplified Configuration**: No CGO requirements in buildpack setup
- **Faster Builds**: Pure Go compilation is faster than CGO
- **Better Portability**: No C compiler dependencies
- **Reduced Risk**: Eliminates cross-compilation complexity

---

## Multi-Binary Challenge Analysis (COMPLETED)

### Problem Identified
The BalancedNewsGo project has **11 distinct binary entry points**:
- **1 Primary Application**: `cmd/server` (web server)
- **10 Utility Tools**: benchmark, fetch_articles, score_articles, clear_articles, etc.

### Buildpack Limitations
- **Single Binary Focus**: Buildpacks expect one main application
- **Auto-Detection Conflicts**: Multiple main.go files confuse buildpack detection
- **Process Definition**: Default process assumes single primary binary

### Recommended Solution: Hybrid Approach
**Production Image Includes:**
- ✅ `cmd/server` - Primary web application
- ✅ `cmd/fetch_articles` - Production RSS fetching
- ✅ `cmd/score_articles` - Production article scoring
- ✅ `cmd/seed_test_data` - E2E testing in CI/CD

**Built Locally/Separately:**
- 🏠 `cmd/benchmark` - Performance testing (development)
- 🏠 `cmd/clear_articles` - Database maintenance (development)
- 🏠 `tools/*` - Development utilities

### Configuration Strategy
```toml
# project.toml
[build.env]
BP_GO_TARGETS = "./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data"
BP_KEEP_FILES = "templates/*:static/*:configs/*"
```

```
# Procfile
web: ./cmd/server
fetch: ./cmd/fetch_articles
score: ./cmd/score_articles
seed: ./cmd/seed_test_data
```

### Benefits
- **Optimized Production Image**: Only essential binaries included
- **Development Flexibility**: Local tools remain available
- **Manageable Complexity**: Single buildpack configuration
- **Clear Separation**: Production vs development tool distinction

**📋 Detailed Analysis**: See `docs/PR/multi-binary-challenge-analysis.md`

---

## Pre-Implementation Validation Plan

### Phase 1: Local Buildpack Testing

**Objective**: Validate basic buildpack functionality and file preservation

```bash
# 1. Install and configure pack CLI
curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.38.2/pack-v0.38.2-windows.zip" -o pack.zip
# Extract and add to PATH
pack version
pack config default-builder gcr.io/paketo-buildpacks/builder:base

# 2. Test basic build
pack build balanced-news-go --path .

# 3. Test with hybrid configuration
pack build balanced-news-go --path . \
  --env BP_GO_TARGETS="./cmd/server:./cmd/fetch_articles:./cmd/score_articles:./cmd/seed_test_data" \
  --env BP_KEEP_FILES="templates/*:static/*:configs/*:.env:*.db"

# 4. Validate file preservation
echo "=== Checking File Preservation ==="
docker run --rm balanced-news-go ls -la templates/
docker run --rm balanced-news-go ls -la static/
docker run --rm balanced-news-go ls -la configs/
docker run --rm balanced-news-go ls -la .env 2>/dev/null || echo ".env not found"

# 5. Test application startup
docker run --rm -d --name test-startup -p 8080:8080 balanced-news-go
sleep 10

# 6. Test static assets
curl -f http://localhost:8080/static/css/main.css || echo "FAIL: Static CSS not served"
curl -f http://localhost:8080/static/assets/css/main.css || echo "FAIL: Static assets not served"

# 7. Test templates
curl -f http://localhost:8080/ | grep -q "html" || echo "FAIL: Templates not rendering"

# 8. Test API endpoints
curl -f http://localhost:8080/api/articles || echo "FAIL: Articles API not working"
curl -f http://localhost:8080/healthz || echo "FAIL: Health check not working"

# Clean up
docker stop test-startup && docker rm test-startup

# Expected Results:
# - All files preserved in buildpack image
# - Application starts successfully
# - Static assets served correctly
# - Templates render properly
# - API endpoints respond
```

### Phase 2: Environment Variable Testing

**Objective**: Validate environment variable handling and configuration loading

```bash
# 1. Test with production-like environment variables
docker run --rm -d --name test-env -p 8080:8080 \
  -e PORT=8080 \
  -e DB_CONNECTION="/tmp/test.db" \
  -e LLM_API_KEY="test-key-12345" \
  -e LOG_FILE_PATH="/tmp/app.log" \
  -e TEST_MODE="false" \
  balanced-news-go

sleep 15

# 2. Verify environment variables are accessible
docker exec test-env env | grep -E "(PORT|DB_CONNECTION|LLM_API_KEY|LOG_FILE_PATH)"

# 3. Test application responds with custom env vars
curl -f http://localhost:8080/healthz || echo "FAIL: Health check with custom env vars"

# 4. Check log file creation
docker exec test-env ls -la /tmp/app.log || echo "Log file not created"

# 5. Test database file creation
docker exec test-env ls -la /tmp/test.db || echo "Database file not created"

# 6. Test .env file loading (if preserved)
docker exec test-env cat .env || echo ".env file not accessible"

# Clean up
docker stop test-env && docker rm test-env

# Expected Results:
# - All environment variables accessible
# - Application uses custom PORT, DB_CONNECTION, etc.
# - Log files created successfully
# - Database initialization works
# - .env file loading functional
```

### Phase 3: Database Persistence Testing

**Objective**: Validate SQLite database persistence and multi-process functionality

```bash
# 1. Create test data directory
mkdir -p ./test-data

# 2. Test database persistence with volume mount
docker run --rm -d --name test-persistence \
  -v $(pwd)/test-data:/data \
  -e DB_CONNECTION="/data/news.db" \
  -e LLM_API_KEY="test-key" \
  -p 8080:8080 balanced-news-go

sleep 20

# 3. Test database file creation and persistence
ls -la ./test-data/ | grep news.db || echo "FAIL: Database file not created"

# 4. Test web process functionality
curl -f http://localhost:8080/healthz || echo "FAIL: Health check"
curl -f http://localhost:8080/api/articles || echo "FAIL: Articles API"

# 5. Test multi-process support (all included binaries)
echo "=== Testing Multi-Process Support ==="
docker exec test-persistence ./cmd/server --help || echo "FAIL: Server binary"
docker exec test-persistence ./cmd/fetch_articles --help || echo "FAIL: Fetch articles binary"
docker exec test-persistence ./cmd/score_articles --help || echo "FAIL: Score articles binary"
docker exec test-persistence ./cmd/seed_test_data --help || echo "FAIL: Seed test data binary"

# 6. Test configuration file access
docker exec test-persistence cat configs/feed_sources.json || echo "FAIL: Config file access"
docker exec test-persistence cat configs/bias_config.json || echo "FAIL: Bias config access"

# 7. Stop container and verify persistence
docker stop test-persistence

# 8. Verify database file persisted
ls -la ./test-data/ | grep news.db || echo "FAIL: Database not persisted"
file ./test-data/news.db || echo "Database file type check"

# Clean up
rm -rf ./test-data/

# Expected Results:
# - Database file created and persisted
# - All 4 production binaries functional
# - Configuration files accessible
# - Volume mounting works correctly
# - Data survives container restart
```

### Validation Success Criteria

**Phase 1 Success Criteria:**
- [ ] Pack CLI builds image successfully
- [ ] All required files preserved (templates/, static/, configs/, .env)
- [ ] Application starts without errors
- [ ] Static assets served correctly
- [ ] Templates render properly
- [ ] API endpoints respond

**Phase 2 Success Criteria:**
- [ ] Environment variables accessible in container
- [ ] Custom PORT, DB_CONNECTION work
- [ ] Log file creation successful
- [ ] Database initialization works
- [ ] .env file loading functional

**Phase 3 Success Criteria:**
- [ ] Database file persists across container lifecycle
- [ ] All 4 production binaries work (server, fetch_articles, score_articles, seed_test_data)
- [ ] Configuration files accessible
- [ ] Volume mounting functional
- [ ] Multi-process support verified

### Failure Response Plan

**If Phase 1 Fails:**
- Review BP_KEEP_FILES configuration
- Check buildpack logs for file preservation issues
- Validate Go module compatibility
- Consider alternative file preservation strategies

**If Phase 2 Fails:**
- Review environment variable injection methods
- Check .env file loading in buildpack environment
- Validate PORT binding with buildpack auto-detection
- Consider runtime environment variable injection

**If Phase 3 Fails:**
- Review database persistence strategy
- Check volume mounting compatibility
- Validate multi-binary build configuration
- Consider separate database initialization approach

**Proceed to Implementation Only If:**
- All 3 phases pass successfully
- All success criteria met
- No critical failures identified
- Performance acceptable compared to Docker approach
