# Go / tools
GO              := go
GOLANGCI        := golangci-lint
SWAG            := swag

# Paths
SERVER_CMD      := ./cmd/server/main.go
BIN_DIR         := ./bin
COVER_DIR       := ./coverage
MOCK_GO         := ./tools/mock_llm_service.go
MOCK_PY         := ./tools/mock_llm_service.py

# Binaries
SERVER_BIN      := $(BIN_DIR)/newbalancer_server

# Flags
RACE            := -race -count=1
SHORT           := -short

.PHONY: help build run clean \
        tidy lint unit integ e2e test coverage-core \
        mock-llm-go mock-llm-py docker-up docker-down

.DEFAULT_GOAL := help

help: ## Show this help
	@awk 'BEGIN {FS=":.*##"; printf "\nTargets:\n"} \
	     /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' \
	     $(MAKEFILE_LIST)

# Build / Run / Clean
# ===================

$(BIN_DIR):
	@mkdir -p $@

build: $(BIN_DIR) ## Compile backend server into ./bin
	$(GO) build -o $(SERVER_BIN) $(SERVER_CMD)

run: build ## Build and run server
	$(SERVER_BIN)

clean: ## Remove binaries & coverage artefacts
	rm -rf $(BIN_DIR) $(COVER_DIR) coverage*.out

# Code Quality
# ==============

tidy: ## go mod tidy
	$(GO) mod tidy

lint: ## Run golangci-lint
	$(GOLANGCI) run ./...

# Testing Matrix
# ===============

unit: ## Fast unit tests
	$(GO) test $(RACE) ./... -run . $(SHORT) -timeout 2m

integ: ## Go integration tests (requires DB etc.)
	$(GO) test -tags=integration ./...

e2e: docker-up ## Docker stack + Playwright e2e
	pnpm --filter=web test:e2e
	$(MAKE) docker-down

test: unit integ ## Run all Go tests

# Coverage Enforcement
# =====================

$(COVER_DIR):
	@mkdir -p $@

coverage-core: $(COVER_DIR) ## Ensure ≥90 % on core pkgs
	@echo "→ Core coverage check"
	@$(GO) test -coverpkg=./internal/llm,./internal/db,./internal/api \
	    -coverprofile=$(COVER_DIR)/core.out ./internal/llm ./internal/db ./internal/api
	@$(GO) tool cover -func=$(COVER_DIR)/core.out | \
	    awk '/total:/ {sub(/%/,"",$3); if($$3<90){printf("FAIL %.1f%% < 90%%\n",$$3);exit 1}else{printf("PASS %.1f%% ≥ 90%%\n",$$3)}}'

# Docs & Contract
# =================

docs: ## Generate swagger docs
	$(SWAG) init -g $(SERVER_CMD) -o internal/api/docs --openapi 3

contract: docs ## Lint & diff OpenAPI spec
	npx @stoplight/spectral-cli lint internal/api/docs/swagger.json
	oasdiff breaking internal/api/docs/swagger.json docs_baseline/swagger.json

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