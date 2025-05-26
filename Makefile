# Attempt to use bash/sh for shell-specific syntax
SHELL = sh

# Configurable race detection flag (default: enabled)
ENABLE_RACE_DETECTION ?= true

# OS detection
ifeq ($(OS),Windows_NT)
    OS_DETECTED := Windows
else
    OS_DETECTED := $(shell uname -s)
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

# Add race flag conditionally
ifeq ($(ENABLE_RACE_DETECTION),true)
    GO_TEST_CMD += -race
endif

# Go / tools
GO              := go
GOLANGCI        := golangci-lint
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
SERVER_BIN      := $(BIN_DIR)/newbalancer_server

# Flags
SHORT           := -short

.PHONY: help build run clean \
        tidy lint unit integ e2e test coverage-core \
        mock-llm-go mock-llm-py docker-up docker-down

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
	$(GO) build -o $(SERVER_BIN) ./cmd/server/...

run: build ## Build and run server
	$(SERVER_BIN)

clean:
	@echo "Cleaning build artifacts..."
	@go run ./tools/clean/main.go
	@echo "Clean complete."

# Code Quality
# ==============

tidy: ## go mod tidy
	$(GO) mod tidy

lint: ## Run golangci-lint
	$(GOLANGCI) run ./...

# Testing Matrix
# ===============

unit:
	@echo "Running unit tests..."
	@echo "OS detected: $(OS_DETECTED)"
ifeq ($(ENABLE_RACE_DETECTION),true)
	@echo "Race detection enabled. Ensure you have a C compiler installed."
	@echo "On Windows, you need MinGW-w64 or another C compiler in your PATH."
	@echo "To disable race detection: make unit ENABLE_RACE_DETECTION=false"
endif
	$(GO_TEST_CMD)
	@echo "Unit tests complete."

integ: ## Go integration tests (requires DB etc.)
	$(GO) test -tags=integration ./cmd/... ./internal/...

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