# PR Plan: Fix main.go RegisterRoutes Signature Error and Align with Refactored API

## Context

A recent refactor (see: "Refactoring API Handlers for Testability") updated the `api.RegisterRoutes` function signature to require two new dependencies: `*llm.ProgressManager` and `*api.SimpleCache`. The main application entrypoint (`cmd/server/main.go`) has not yet been updated to provide these arguments, resulting in a build error:

```
not enough arguments in call to api.RegisterRoutes
have (*gin.Engine, *sqlx.DB, *...Collector, *llm.LLMClient, *llm.ScoreManager)
want (*gin.Engine, *sqlx.DB, ...CollectorInterface, *llm.LLMClient, *llm.ScoreManager, *llm.ProgressManager, *api.SimpleCache)
```

This plan details the steps required to resolve this error, ensure all dependencies are properly instantiated, and align the main application with the new API handler structure.

---

## Step-by-Step Plan

### 1. Review the New RegisterRoutes Signature
- **Location:** `internal/api/api.go`
- **New Signature:**
  ```go
  func RegisterRoutes(
      router *gin.Engine,
      dbConn *sqlx.DB,
      rssCollector rss.CollectorInterface,
      llmClient *llm.LLMClient,
      scoreManager *llm.ScoreManager,
      progressManager *llm.ProgressManager,
      cache *api.SimpleCache,
  )
  ```
- **Required:**
  - `progressManager` (for tracking LLM scoring progress)
  - `cache` (for API-level caching)

### 2. Update main.go to Provide All Required Arguments
- **File:** `cmd/server/main.go`
- **Current Call:**
  ```go
  api.RegisterRoutes(router, dbConn, rssCollector, llmClient, scoreManager)
  ```
- **Required Additions:**
  - Instantiate a `*llm.ProgressManager` (see `llm.NewProgressManager`)
  - Instantiate a `*api.SimpleCache` (see `api.NewSimpleCache`)
  - Pass both to `RegisterRoutes`.

#### 2.1. Instantiate ProgressManager
- **Import:** `"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"`
- **Construction:**
  ```go
  progressManager := llm.NewProgressManager(time.Minute) // 1-minute cleanup interval (adjust as needed)
  ```
- **Rationale:**
  - Handles progress tracking and cleanup for LLM scoring jobs.
  - The interval can be tuned; 1 minute is a reasonable default.

#### 2.2. Instantiate SimpleCache
- **Import:** `"github.com/alexandru-savinov/BalancedNewsGo/internal/api"`
- **Construction:**
  ```go
  cache := api.NewSimpleCache()
  ```
- **Rationale:**
  - Provides in-memory caching for API responses (articles, summaries, etc).

#### 2.3. Update RegisterRoutes Call
- **New Call:**
  ```go
  api.RegisterRoutes(router, dbConn, rssCollector, llmClient, scoreManager, progressManager, cache)
  ```

### 3. Test and Validate
- **Build:** Ensure `go build ./...` succeeds.
- **Run:** Start the server (`go run cmd/server/main.go`) and verify it launches without error.
  - **Immediate Check:** This is the very first "test." The server starting without the "not enough arguments" error directly confirms the `api.RegisterRoutes` signature mismatch is resolved.
- **Test:** Run all test suites (see README for scripts) to confirm no regressions.
  - **Specific Go Tests to Verify Integration:**
    - **Run all tests in the `internal/api` package:**
      ```bash
      go test ./internal/api -v
      ```
      - **Rationale:** This suite is crucial as it tests the API handlers where `ProgressManager` and `SimpleCache` are utilized. It includes integration tests (e.g., in `internal/api/api_integration_test.go`) that mock dependencies and verify interactions with caching and progress tracking (e.g., `TestSSEProgressEndpointIntegration`, `TestManualScoreCacheInvalidation`).
    - **Optional: Run individual test functions for targeted checks:**
      ```bash
      go test ./internal/api -v -run TestSSEProgressEndpointIntegration
      go test ./internal/api -v -run TestManualScoreCacheInvalidation
      ```
      - **Rationale:** Useful if you want to focus on a specific area related to the new dependencies after initial broader tests.
- **Manual:** Optionally, hit API endpoints to confirm caching and progress tracking work as expected.

### 4. Documentation and Code Comments
- **Document** the rationale for the new dependencies in `main.go`.
- **Add comments** explaining the choice of cleanup interval for `ProgressManager` and the use of `SimpleCache`.

### 5. Reference: Previous Refactor
- The changes are a direct follow-up to the "Refactoring API Handlers for Testability" work, which moved API handler dependencies to explicit arguments for better testability and modularity.
- See also: `internal/api/api.go`, `internal/llm/progress_manager.go`, `internal/api/cache.go` for implementation details.

---

## Checklist
- [x] Instantiate `ProgressManager` in `main.go`
- [x] Instantiate `SimpleCache` in `main.go`
- [x] Pass both to `api.RegisterRoutes`
- [x] Build and run the server successfully
- [x] Run all tests and confirm passing
- [x] Update documentation/comments as needed

---

## Notes
- If additional dependencies are required by `ScoreManager` or other components, ensure they are also constructed and passed as needed.
- If the API or handler signatures change again, update this plan accordingly. 
- **Suggestion (Optional):** For very large files, consider noting approximate line numbers for the `api.RegisterRoutes` call in `main.go` and its definition in `api.go` for slightly faster navigation, though modern IDEs usually make this less critical.
- **Suggestion (Consideration):** If there were any subtle initialization order dependencies between the new `ProgressManager`, `SimpleCache`, and existing services (like `ScoreManager`), it would be worth a brief mention. Currently, they appear independent. 