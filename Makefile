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

# Flags
SHORT           := -short

.PHONY: help build run stop restart clean \
        tidy lint unit integ e2e test coverage-core coverage coverage-html \
        mock-llm-go mock-llm-py docker-up docker-down integration benchmark \
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

# Code Quality
# ==============

tidy: ## go mod tidy
	$(GO) mod tidy

lint: ## Run golangci-lint
	$(GOLANGCI) run --timeout=5m

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

# Docker-Compose Helpers
# ========================

docker-up: ## Spin up full Docker stack
	docker compose -f infra/docker-compose.yml up -d

docker-down: ## Tear down Docker stack
	docker compose -f infra/docker-compose.yml down -v

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
	@echo "✓ Running go vet..."
	@$(GO) vet -composites=false ./...
	@echo "✓ Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "⚠ staticcheck not found, skipping (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi
	@echo ""
	@echo "🏗️ Step 3: Build verification..."
	@echo "✓ Building application..."
	@$(GO) build -o test-server-precommit$(if $(filter Windows_NT,$(OS)),.exe,) ./cmd/server
	@rm -f test-server-precommit$(if $(filter Windows_NT,$(OS)),.exe,)
	@echo ""
	@echo "🧪 Step 4: Running unit tests..."
	@echo "✓ Running unit tests..."
	@NO_AUTO_ANALYZE=true NO_DOCKER=true $(MAKE) unit ENABLE_RACE_DETECTION=false
	@echo ""
	@echo "🎉 All pre-commit checks passed! Your changes are ready to commit."
	@echo ""
	@echo "Note: This runs the same checks as the CI/CD pipeline."
	@echo "If this passes, your CI build should also pass."
