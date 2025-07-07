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
BP_GO_BUILD_LDFLAGS = "-w -s"
CGO_ENABLED = "0"
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

Task 3: Configure Multi-Process Support
  - CREATE Procfile for different process types (web, worker, benchmark)
  - DEFINE web process for main server application
  - DEFINE benchmark process for performance testing
  - TEST each process type builds and runs correctly

Task 4: Handle Static Assets and Templates
  - ENSURE static/ and templates/ directories are included in build
  - CONFIGURE buildpack to preserve directory structure
  - TEST web application serves static assets correctly
  - VALIDATE template rendering works

Task 5: Update Build Scripts and Makefile
  - MODIFY Makefile to use pack commands instead of docker
  - UPDATE build targets for different environments
  - CREATE development build with live reload capability
  - TEST all make targets work with buildpacks

Task 6: Update CI/CD Pipeline
  - MODIFY .github/workflows/ci.yml to use pack CLI
  - REPLACE docker build steps with pack build
  - UPDATE image registry push commands
  - TEST CI/CD pipeline builds successfully

Task 7: Remove Docker Configuration
  - DELETE Dockerfile and Dockerfile.app
  - REMOVE docker-related scripts and configurations
  - CLEAN UP docker references in documentation
  - VERIFY no docker dependencies remain

Task 8: Update Documentation
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
CGO_ENABLED = "0"
EOF

# Test basic build
pack build balanced-news-go --path .

# Task 3: Multi-Process Configuration
# Create Procfile for different process types
cat > Procfile << EOF
web: ./cmd/server/main
benchmark: ./cmd/benchmark/main
EOF

# Build with specific process
pack build balanced-news-go:web --path . --env BP_GO_TARGETS="./cmd/server"
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

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests - Ensure buildpack preserves functionality
```bash
# Test basic build
pack build balanced-news-go --path .

# Test application starts
docker run --rm -p 8080:8080 balanced-news-go &
sleep 5
curl -f http://localhost:8080/api/articles || exit 1

# Test static assets
curl -f http://localhost:8080/static/css/main.css || exit 1

# Test templates
curl -f http://localhost:8080/ | grep -q "html" || exit 1

# Clean up
docker stop $(docker ps -q --filter ancestor=balanced-news-go)
```

### Level 3: Integration Test
```bash
# Build all process types
pack build balanced-news-go:web --path . --env BP_GO_TARGETS="./cmd/server"
pack build balanced-news-go:benchmark --path . --env BP_GO_TARGETS="./cmd/benchmark"

# Test web process
docker run --rm -d -p 8080:8080 --name test-web balanced-news-go:web
sleep 10
curl -f http://localhost:8080/api/articles
docker stop test-web

# Test benchmark process
docker run --rm balanced-news-go:benchmark --help

# Expected: All processes build and run successfully
```

## Final validation Checklist
- [ ] Pack CLI installed and configured: `pack version`
- [ ] Basic build works: `pack build balanced-news-go --path .`
- [ ] Application starts: `docker run balanced-news-go`
- [ ] Web endpoints respond: `curl http://localhost:8080/api/articles`
- [ ] Static assets served: `curl http://localhost:8080/static/css/main.css`
- [ ] Templates render: `curl http://localhost:8080/`
- [ ] Benchmark builds: `pack build balanced-news-go:benchmark`
- [ ] CI/CD pipeline passes: GitHub Actions green
- [ ] Docker files removed: `ls Dockerfile*` returns nothing
- [ ] Documentation updated: README.md reflects buildpack usage

---

## Risk Assessment and Rollback Procedures

### High Risk Areas
1. **✅ Pure Go SQLite**: VERIFIED - All files now consistently use modernc.org/sqlite (pure Go)
2. **Static Asset Serving**: Templates and static files must be preserved in build
3. **Multi-Binary Builds**: Different entry points (server, benchmark, tools) need separate builds
4. **CI/CD Integration**: GitHub Actions must be updated without breaking deployment

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

## PRP Confidence Score: 9/10

**Reasoning for 9/10 Score:**
- **Very High Confidence (9)**: Buildpacks are mature technology with strong Go support
- **Well-Defined Requirements**: Clear understanding of current Docker setup and needs
- **Proven Technology**: CNB is CNCF project with production usage
- **Good Documentation**: Extensive buildpack documentation and examples available
- **✅ Validated Pure Go**: All SQLite drivers now consistently use pure Go (no CGO)
- **✅ Test Coverage**: Comprehensive test suite passes with pure Go drivers

**Risk Factors (-1 point):**
- **Multi-Binary Challenge**: Multiple entry points may need custom buildpack configuration
- **Static Asset Handling**: Need to ensure templates/static files are properly included

**Risk Mitigation Completed:**
- **✅ Driver Consistency**: All SQLite drivers now consistently use pure Go (modernc.org/sqlite)
- **✅ Test Validation**: All unit, integration, and E2E tests pass with pure Go driver
- **✅ Build Verification**: Main application builds and runs correctly

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
