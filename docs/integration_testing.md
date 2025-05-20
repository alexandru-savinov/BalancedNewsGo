# Integration Testing Guide

This guide focuses on performing integration testing with the NewsBalancer system, particularly for components that interact with external services like LLM APIs.

## Prerequisites

Before running integration tests, ensure you have:

- Go 1.19+ installed
- SQLite3 command-line tools (optional, for database inspection)
- Newman/Postman for API testing
- Valid LLM API credentials (or mock configuration)
- Test database properly set up

## Environment Setup

For reliable integration testing, set these environment variables:

```bash
# Windows PowerShell
$env:NO_AUTO_ANALYZE='true'  # Prevents background LLM analysis during tests
$env:LLM_API_KEY='your_api_key'  # Use a test key with limited quota

# Linux/macOS
export NO_AUTO_ANALYZE=true
export LLM_API_KEY=your_api_key
```

## Using the Mock LLM Service

The mock LLM service (`mock_llm_service.go`) provides a way to test without calling actual LLM APIs, saving costs and providing deterministic behavior.

### Starting the Mock Service

```bash
# Start with default configuration (returns fixed scores)
go run mock_llm_service.go

# Start with specific port
go run mock_llm_service.go -port 8090

# Start with specific response delay and custom scoring
go run mock_llm_service.go -delay 500 -score 0.25 -label "center-right"
```

### Configuring the System to Use the Mock

Update your `.env` file or environment variables:

```
LLM_BASE_URL=http://localhost:8090  # Match the port of your mock service
```

### Mock Service Endpoints

The mock service exposes these endpoints:

- `POST /analyze` - Returns mock political bias analysis
- `GET /status` - Health check and configuration display
- `POST /configure` - Dynamically change mock behavior

### Mock Service Response Format

The service returns responses in this format:

```json
{
  "score": 0.25,
  "confidence": 0.85,
  "label": "center-right",
  "explanation": "This is a mock analysis response"
}
```

## Integration Test Suites

The system includes several integration test suites:

### Backend Integration Tests

```bash
# Windows
scripts/test.cmd backend

# Linux/macOS
scripts/test.sh backend
```

This executes the Newman collection `postman/backend_tests.json` which tests core backend functionality by making actual API calls to a running server.

### API Tests

```bash
# Windows
scripts/test.cmd api

# Linux/macOS
scripts/test.sh api
```

Executes Newman collection `postman/api_tests.json` to verify API contracts and responses.

## Testing Individual Components

### Testing the LLM Integration

Use `cmd/test_llm/main.go` to test LLM API integration:

```bash
go run cmd/test_llm/main.go
```

This sends a predefined article to the configured LLM service and displays the response.

### Testing the RSS Feed Collection

```bash
go run cmd/fetch_articles/main.go
```

This triggers RSS feed collection without starting the full server.

## Creating Custom Test Data

### Adding Test Articles

Use `internal/testing.InsertTestArticle` in your test code:

```go
import "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"

func TestSomething(t *testing.T) {
    db := setupTestDB(t)
    articleID, err := testing.InsertTestArticle(db, "Test Title", "Test Content")
    // Continue test with articleID
}
```

### Adding Test LLM Scores

Use `internal/testing.SeedLLMScoresForSuccessfulScore`:

```go
import "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"

func TestSomething(t *testing.T) {
    db := setupTestDB(t)
    articleID, _ := testing.InsertTestArticle(db, "Test Title", "Test Content")
    err := testing.SeedLLMScoresForSuccessfulScore(db, articleID)
    // Continue test with fully scored article
}
```

## Handling Test Database

The test framework creates an in-memory SQLite database by default. For persistent tests, you can:

```bash
# Reset the test database (PowerShell)
./recreate_db.ps1  # Windows

# Manually clear specific test data or recreate the DB on other platforms
go run cmd/reset_test_db/main.go
```

## Diagnosing Integration Test Failures

Common issues and solutions:

1. **Database Constraint Errors**: Ensure the `llm_scores` table has the `UNIQUE(article_id, model)` constraint.

2. **Port Conflicts**: Check if port 8080 is already in use:
   ```bash
   # Windows
   Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue | 
      Select-Object -ExpandProperty OwningProcess | 
      ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }
   
   # Linux/macOS
   lsof -i :8080 | awk 'NR>1 {print $2}' | xargs kill -9
   ```

3. **Mock LLM Service Issues**: Verify the mock service is running and the base URL is properly configured.

4. **LLM API Errors**: Check the API key validity and ensure it has sufficient quota.

5. **Database Locks**: Use `NO_AUTO_ANALYZE=true` and ensure no background processes are accessing the database.

## Performance Testing

For performance testing of LLM integrations:

1. Use the `LLM_HTTP_TIMEOUT` environment variable to test timeout handling
2. Benchmark LLM response times with mock service delays:
   ```bash
   go run mock_llm_service.go -delay 2000  # 2 second response delay
   ```
3. Test high concurrency with the mock service:
   ```bash
   go test -bench=BenchmarkConcurrentLLMCalls -benchtime=10s ./internal/llm/...
   ```

## Continuous Integration

For CI pipelines, always use:
- `NO_AUTO_ANALYZE=true`
- The mock LLM service
- Isolated database instances
- Explicit port assignments (to avoid conflicts) 