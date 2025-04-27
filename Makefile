.PHONY: tidy lint test unit docs contract int e2e coverage-core

tidy:          ## keep go.mod clean
	go mod tidy

lint:          ## static analysis
	golangci-lint run ./...

unit:          ## fast, in-memory tests with race detector
	go test -race -count=1 ./... -run . -short

docs:          ## generate OpenAPI 3.1
	swag init -g cmd/server/main.go -o internal/api/docs --openapi 3

contract: docs ## lint the spec & check breaking changes
	npx @stoplight/spectral-cli lint internal/api/docs/swagger.json
	oasdiff breaking internal/api/docs/swagger.json docs_baseline/swagger.json

int: contract  ## spin real handler; hit it via httpexpect
	go test -tags=integration ./...

e2e: contract  ## full Docker stack + Playwright
	docker compose -f infra/docker-compose.yml up -d
	pnpm --filter=web test:e2e

coverage-core:  ## measure coverage on core packages and enforce >=90%
	@echo "Running core coverage check (internal/llm, internal/db, internal/api)"
	@go test -coverpkg=./internal/llm,./internal/db,./internal/api \
		-coverprofile=coverage-core.out ./internal/llm ./internal/db ./internal/api
	@go tool cover -func=coverage-core.out | grep total: | awk '{print $$3}' | sed 's/%//' | \
		awk '{ if ($$1 < 90) { printf("Core coverage %.1f%% < 90%%\n", $$1); exit 1 } else { printf("Core coverage %.1f%% >= 90%%\n", $$1) } }'