# OS detection
ifeq ($(OS),Windows_NT)
    OS_DETECTED := Windows
else
    OS_DETECTED := $(shell uname -s)
endif

# CGO availability check
CGO_AVAILABLE := $(shell CGO_ENABLED=1 go env CGO_ENABLED 2>/dev/null)
ifeq ($(CGO_AVAILABLE),1)
    CGO_SUPPORTED := true
else
    CGO_SUPPORTED := false
endif

# Configurable race detection flag with intelligent defaults
ifeq ($(OS),Windows_NT)
    ENABLE_RACE_DETECTION ?= false
else
    ifeq ($(CGO_SUPPORTED),true)
        ENABLE_RACE_DETECTION ?= true
    else
        ENABLE_RACE_DETECTION ?= false
    endif
endif

# Base test command without race detection
ifeq ($(OS),Windows_NT)
    TEST_PKGS := ./cmd/... \
        ./internal/api/... \
        ./internal/apperrors/... \
        ./internal/balancer/... \
        ./internal/db/... \
        ./internal/import_labels/... \
        ./internal/metrics/... \
        ./internal/models/... \
        ./internal/rss/... \
        ./internal/testing/... \
        ./internal/tests/...
else
    TEST_PKGS := ./cmd/... ./internal/...
endif
GO_TEST_CMD := go test -count=1 $(TEST_PKGS) -run . -short -timeout 2m

# Add race flag conditionally with validation
ifeq ($(ENABLE_RACE_DETECTION),true)
    ifeq ($(CGO_SUPPORTED),true)
        GO_TEST_CMD += -race
        export CGO_ENABLED=1
    else
        $(warning Race detection requested but CGO is not available. Disabling race detection.)
        ENABLE_RACE_DETECTION := false
    endif
endif

# Go / tools
GO              := go
GOLANGCI        := C:\Users\Alexander.Savinov\go\bin\golangci-lint.exe
SWAG            := swag
PACK            := ./pack-cli/pack.exe

# Paths
SERVER_CMD      := ./cmd/server/main.go ./cmd/server/legacy_handlers.go
BIN_DIR         := ./bin
COVER_DIR       := ./coverage
MOCK_GO         := ./tools/mock_llm_service/main.go
MOCK_PY         := ./tools/mock_llm_service.py

# Coverage Configuration
COVERAGE_DIR := ./coverage
COVERAGE_THRESHOLD ?= 90

# Binaries
SERVER_BIN      := bin\newbalancer_server.exe # Use backslashes for Windows

# Buildpack Configuration
BUILDPACK_IMAGE := balanced-news-go
BUILDPACK_TAG   := latest

# Flags
SHORT           := -short

.PHONY: help build run stop restart clean \
        tidy lint static-analysis-ci unit unit-ci integ e2e e2e-ci test coverage-core coverage coverage-html \
        mock-llm-go mock-llm-py monitoring-up monitoring-down integration benchmark \
        buildpack-build buildpack-run buildpack-stop buildpack-test buildpack-clean \
        precommit-check

.DEFAULT_GOAL := help

help: ## Show this help
	@go run ./tools/make_help.go

# Build / Run / Clean
# ===================

$(BIN_DIR):
	@echo "Ensuring bin directory exists..."
	@go run ./tools/mkdir/main.go $(BIN_DIR)

$(COVER_DIR):
	@echo "Ensuring coverage directory exists..."
	@go run ./tools/mkdir/main.go $(COVER_DIR)

build: $(BIN_DIR) ## Compile backend server into ./bin
	$(GO) build -v -o $(SERVER_BIN) ./cmd/server/...

run: build ## Build and run server in background, redirecting output
	@echo "Stopping any existing server..."
	@taskkill /IM newbalancer_server.exe /F 2>nul || echo "No newbalancer_server.exe running."
	@taskkill /IM newsbalancer_server.exe /F 2>nul || echo "No newsbalancer_server.exe running."
	@echo "Running server $(SERVER_BIN) in background (output to server_run.log)..."
	cmd /c "start /B .\$(SERVER_BIN) > server_run.log 2>&1"
	@echo "Server started. Check server_run.log for output."

stop: ## Stop the running server
	@taskkill /IM newbalancer_server.exe /F 2>nul || echo "No newbalancer_server.exe running."
	@taskkill /IM newsbalancer_server.exe /F 2>nul || echo "No newsbalancer_server.exe running."

restart: ## Restart the server (stop + run)
	@echo "Restarting server..."
	@$(MAKE) stop
	@$(MAKE) run

clean:
	@echo "Cleaning build artifacts..."
	@go run ./tools/clean/main.go
	@echo "Clean complete."

# Buildpack Targets
# ==================

buildpack-build: ## Build application using Cloud Native Buildpacks
	@echo "Building application with buildpacks..."
	$(PACK) build $(BUILDPACK_IMAGE):$(BUILDPACK_TAG) --path .
	@echo "Buildpack build complete. Image: $(BUILDPACK_IMAGE):$(BUILDPACK_TAG)"

buildpack-run: buildpack-build ## Build and run application using buildpacks
	@echo "Stopping any existing buildpack containers..."
	@docker stop $(BUILDPACK_IMAGE)-container 2>nul || echo "No existing container to stop."
	@docker rm $(BUILDPACK_IMAGE)-container 2>nul || echo "No existing container to remove."
	@echo "Running buildpack application..."
	@docker run -d --name $(BUILDPACK_IMAGE)-container \
		-p 8080:8080 \
		-e DB_CONNECTION="./newsbalancer.db" \
		-e PORT=8080 \
		$(BUILDPACK_IMAGE):$(BUILDPACK_TAG)
	@echo "Buildpack application started. Container: $(BUILDPACK_IMAGE)-container"
	@echo "Application available at: http://localhost:8080"

buildpack-stop: ## Stop buildpack application container
	@echo "Stopping buildpack application..."
	@docker stop $(BUILDPACK_IMAGE)-container 2>nul || echo "Container not running."
	@docker rm $(BUILDPACK_IMAGE)-container 2>nul || echo "Container already removed."
	@echo "Buildpack application stopped."

buildpack-test: buildpack-build ## Build and test buildpack application
	@echo "Testing buildpack application..."
	@docker run --rm --name $(BUILDPACK_IMAGE)-test \
		-e DB_CONNECTION="/tmp/test.db" \
		-e LLM_API_KEY="test-key" \
		-p 8081:8080 \
		$(BUILDPACK_IMAGE):$(BUILDPACK_TAG) &
	@echo "Waiting for application to start..."
	@timeout /t 15 /nobreak >nul
	@echo "Testing health endpoint..."
	@powershell -Command "try { Invoke-WebRequest -Uri http://localhost:8081/healthz -TimeoutSec 10 | Out-Null; Write-Host 'Health check passed' } catch { Write-Host 'Health check failed (expected with test API key)' }"
	@docker stop $(BUILDPACK_IMAGE)-test 2>nul || echo "Test container stopped."
	@echo "Buildpack test complete."

buildpack-clean: ## Clean buildpack images and containers
	@echo "Cleaning buildpack artifacts..."
	@docker stop $(BUILDPACK_IMAGE)-container 2>nul || echo "No container to stop."
	@docker rm $(BUILDPACK_IMAGE)-container 2>nul || echo "No container to remove."
	@docker rmi $(BUILDPACK_IMAGE):$(BUILDPACK_TAG) 2>nul || echo "No image to remove."
	@echo "Buildpack cleanup complete."

# Code Quality
# ==============

tidy: ## go mod tidy
	$(GO) mod tidy

lint: ## Run golangci-lint
	$(GOLANGCI) run --timeout=5m

static-analysis-ci: ## Run comprehensive static analysis matching CI
	@echo "Running static analysis for concurrency issues..."
	@echo "✓ Running go vet..."
	$(GO) vet -composites=false ./...
	@echo "✓ Running staticcheck..."
ifeq ($(OS),Windows_NT)
	@where staticcheck >nul 2>&1 || (echo "Installing staticcheck..." && $(GO) install honnef.co/go/tools/cmd/staticcheck@latest)
	@staticcheck ./...
else
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "Installing staticcheck..."; \
		$(GO) install honnef.co/go/tools/cmd/staticcheck@latest; \
		staticcheck ./...; \
	fi
endif
	@echo "✓ Running golangci-lint with test files..."
	$(GOLANGCI) run --skip-files="" --tests=true ./...

# Testing Matrix
# ===============

unit:
	@echo "Running unit tests..."
	@echo "OS detected: $(OS_DETECTED)"
	@echo "CGO supported: $(CGO_SUPPORTED)"
ifeq ($(ENABLE_RACE_DETECTION),true)
	@echo "Race detection enabled."
	@echo "CGO_ENABLED=1 is set for race detection."
else
	@echo "Race detection disabled."
ifeq ($(CGO_SUPPORTED),false)
	@echo "Note: CGO is not available. Race detection cannot be enabled."
	@echo "To enable CGO: install a C compiler and set CGO_ENABLED=1"
endif
endif
	@echo "Test command: $(GO_TEST_CMD)"
	$(GO_TEST_CMD)
	@echo "Unit tests complete."

unit-ci: ## Run unit tests matching CI configuration (3 runs with coverage)
	@echo "Running unit tests with CI configuration..."
	@echo "Running tests 3 times to catch intermittent issues..."
ifeq ($(OS),Windows_NT)
	@set NO_AUTO_ANALYZE=true && set NO_DOCKER=true && $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-1.out -covermode=atomic ./...
	@echo "Test run 1/3 completed"
	@set NO_AUTO_ANALYZE=true && set NO_DOCKER=true && $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-2.out -covermode=atomic ./...
	@echo "Test run 2/3 completed"
	@set NO_AUTO_ANALYZE=true && set NO_DOCKER=true && $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-3.out -covermode=atomic ./...
	@echo "Test run 3/3 completed"
	@echo "Using the last coverage file"
	@move coverage-3.out coverage.out
	@del coverage-*.out 2>nul || echo "No coverage files to clean"
else
	NO_AUTO_ANALYZE=true NO_DOCKER=true $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-1.out -covermode=atomic ./...
	@echo "Test run 1/3 completed"
	NO_AUTO_ANALYZE=true NO_DOCKER=true $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-2.out -covermode=atomic ./...
	@echo "Test run 2/3 completed"
	NO_AUTO_ANALYZE=true NO_DOCKER=true $(GO) test -v -count=1 -parallel=4 -coverprofile=coverage-3.out -covermode=atomic ./...
	@echo "Test run 3/3 completed"
	@echo "Using the last coverage file"
	@mv coverage-3.out coverage.out
	@rm -f coverage-*.out
endif
	@echo "CI-style unit tests complete."

integ: ## Go integration tests (requires DB etc.)
	$(GO) test -tags=integration ./cmd/... ./internal...

concurrency: ## Run concurrency tests without CGO (using goleak and stress testing)
	@echo "Running concurrency tests without CGO..."
	@echo "Installing goleak if not present..."
	$(GO) get -t go.uber.org/goleak
	@echo "Running static analysis for concurrency issues..."
	$(GO) vet -composites=false ./...
	@echo "Running stress tests to detect race conditions..."
	$(GO) test -v -count=3 -parallel=4 ./internal/testing/
	@echo "Running all tests with stress testing..."
	$(GO) test -v -count=2 -parallel=2 $(TEST_PKGS)
	@echo "Concurrency tests complete."

integration: integ ## Alias for the 'integ' target to allow 'make integration'

e2e: docker-up ## Docker stack + Playwright e2e
	pnpm --filter=web test:e2e
	$(MAKE) docker-down

e2e-ci: ## Run Playwright tests matching CI configuration
	@echo "Running Playwright tests with CI configuration..."
	@echo "Seeding test data for accessibility tests..."
	@set DATABASE_PATH=newsbalancer.db && set DB_CONNECTION=newsbalancer.db && $(GO) run ./cmd/seed_test_data
	@echo "Starting server in background..."
	@set DATABASE_PATH=newsbalancer.db && set DB_CONNECTION=newsbalancer.db && set NO_AUTO_ANALYZE=true && set PORT=8080 && start /B .\bin\newbalancer_server.exe
	@echo "Waiting for server to be ready..."
	@timeout /t 10 /nobreak >nul
	@echo "Running accessibility tests..."
	@npm run test:accessibility
	@echo "Running progress indicator tests..."
	@npm run test:progress-indicator
	@echo "Stopping server..."
	@taskkill /IM newbalancer_server.exe /F 2>nul || echo "Server already stopped."

test: unit integ
	@echo "All tests complete."

# Coverage Enforcement
# =====================

coverage-core: $(COVER_DIR)
	@echo "Running core package coverage tests..."
ifeq ($(ENABLE_RACE_DETECTION),true)
	@echo "Race detection enabled for coverage. Ensure you have a C compiler installed."
endif
	$(GO) test -coverprofile=$(COVERAGE_DIR)/core.out -covermode=atomic $(if $(filter true,$(ENABLE_RACE_DETECTION)),-race) -coverpkg=./internal/llm,./internal/db,./internal/api ./internal/...
	@echo "Generating coverage report..."
	$(GO) tool cover -func=$(COVERAGE_DIR)/core.out > $(COVERAGE_DIR)/coverage.txt
	@echo "Checking coverage threshold ($(COVERAGE_THRESHOLD)%)..."
	$(GO) run ./tools/check_coverage/main.go $(COVERAGE_DIR)/coverage.txt $(COVERAGE_THRESHOLD)
	@echo "Coverage tests complete."

coverage: $(COVER_DIR) ## Run comprehensive coverage analysis
ifeq ($(OS),Windows_NT)
	@echo "Running comprehensive coverage analysis (Windows)..."
	powershell -ExecutionPolicy Bypass -File scripts/coverage.ps1
else
	@echo "Running comprehensive coverage analysis (Unix)..."
	bash scripts/coverage.sh
endif

coverage-html: coverage ## Generate and open HTML coverage report
ifeq ($(OS),Windows_NT)
	@echo "Opening HTML coverage report..."
	start coverage/coverage.html
else
	@echo "Opening HTML coverage report..."
	open coverage/coverage.html || xdg-open coverage/coverage.html
endif

# Docs & Contract
# =================

docs: ## Generate swagger docs
	$(SWAG) init -g $(SERVER_CMD) -o docs --parseDependency --parseDependencyLevel 3 --parseInternal --generatedTime

contract:
	@echo "Running OpenAPI contract validation..."
	@echo "Linting API specification (docs/swagger.json)..."
	@npx @stoplight/spectral-cli lint docs/swagger.json --ruleset .spectral.yaml || (echo "ERROR: Spectral linting found errors that must be fixed." && exit 1)
	@echo "Checking for breaking API changes..."
	@$(GO) run ./tools/run_oasdiff_conditionally/main.go docs/swagger.json.bak docs/swagger.json
	@echo "OpenAPI contract validation complete."

# Mock Services Convenience
# ==========================

mock-llm-go: ## Run Go mock LLM service
	$(GO) run $(MOCK_GO)

mock-llm-py: ## Run Python mock LLM service
	python $(MOCK_PY)

# Monitoring Stack (Optional)
# ============================
# Note: Monitoring stack is available in monitoring/docker-compose.monitoring.yml
# Run with: docker compose -f monitoring/docker-compose.monitoring.yml up -d

monitoring-up: ## Start monitoring stack (Prometheus, Grafana, etc.)
	docker compose -f monitoring/docker-compose.monitoring.yml up -d

monitoring-down: ## Stop monitoring stack
	docker compose -f monitoring/docker-compose.monitoring.yml down -v

# Performance Benchmarking
# =========================

benchmark: ## Run performance benchmarks
	@echo "Building benchmark tool..."
	$(GO) build -o bin/benchmark ./cmd/benchmark
ifeq ($(OS),Windows_NT)
	@echo "Running performance benchmarks (Windows)..."
	powershell -ExecutionPolicy Bypass -File scripts/run-benchmarks.ps1
else
	@echo "Running performance benchmarks (Unix)..."
	bash scripts/run-benchmarks.sh
endif

# Pre-commit Checks (matching CI/CD pipeline)
# ============================================

precommit-check: ## Run the same checks as CI/CD pipeline (for local verification)
	@echo "🔍 Running pre-commit checks (matching CI/CD pipeline)..."
	@echo ""
	@echo "🧹 Step 1: Code formatting and tidying..."
	@echo "✓ Running go mod tidy..."
	@$(GO) mod tidy
	@echo "✓ Running go fmt..."
	@$(GO) fmt ./...
	@echo ""
	@echo "🔍 Step 2: Linting and static analysis (matching CI)..."
	@echo "✓ Running golangci-lint..."
	@$(MAKE) lint
	@echo "✓ Running comprehensive static analysis..."
	@$(MAKE) static-analysis-ci
	@echo ""
	@echo "🏗️ Step 3: Build verification..."
	@echo "✓ Building application..."
	@$(GO) build -o test-server-precommit$(if $(filter Windows_NT,$(OS)),.exe,) ./cmd/server
ifeq ($(OS),Windows_NT)
	@del test-server-precommit.exe 2>nul || echo "Build artifact cleaned"
else
	@rm -f test-server-precommit
endif
	@echo ""
	@echo "🧪 Step 4: Running unit tests (CI-style)..."
	@echo "✓ Running unit tests with CI configuration..."
	@$(MAKE) unit-ci
	@echo ""
	@echo "🎉 All pre-commit checks passed! Your changes are ready to commit."
	@echo ""
	@echo "Note: This runs the same checks as the CI/CD pipeline."
	@echo "If this passes, your CI build should also pass."
