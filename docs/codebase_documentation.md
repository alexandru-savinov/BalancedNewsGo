# Codebase Documentation: NewsBalancer

## 1. Introduction

This document provides a high-level overview and detailed documentation of the Go codebase for the NewsBalancer project. The primary goal of NewsBalancer is to fetch news articles from various RSS feeds, analyze their political bias using Large Language Models (LLMs), store the results, and expose this data through a web interface and API.

**Test Status Update (May 2025):**
- All Go unit, integration, and end-to-end tests now pass, including the previously failing `internal/llm` tests.
- The codebase now uses **averaging everywhere** for duplicate model/perspective scores and confidences. This logic is fully covered by passing tests.
- For reliable test runs, set the environment variable `NO_AUTO_ANALYZE=true` (see `docs/testing.md`).

**High-Level Architecture & Data Flow:**

```
+-----------------+      +----------------------+      +------------------+
| RSS Feeds (Web) | ---> | internal/rss         | ---> | internal/db      |
| (External)      |      | (Collector)          |      | (SQLite Storage) |
+-----------------+      +----------------------+      +--------+---------+
                                                            |
                                                            |  Articles
                                                            v
+-----------------+      +----------------------+      +--------+---------+
| External LLM    | <--> | internal/llm         | <--- | cmd/server       |
| Services (API)  |      | (Client, Ensemble,   |      | internal/api     |
| (e.g.OpenRouter)|      | Score Calculator...) | ---> | (Web UI, REST API)|
+-----------------+      +----------------------+      +--------+---------+
    ^    | Scores                                           |  Scores/Data |
    |    +--------------------------------------------------+  Progress    |
    |                                                       |  Feedback    v
    +-------------------------------------------------------+ User/Client

```

**Core Data Flow Steps:**

1.  **Ingestion:** `internal/rss` fetches articles from configured RSS feeds (`configs/feed_sources.json`).
    *   *Debugging/Improvements: Check feed validity/parsing; add sources; improve content extraction/normalization; enhance duplicate detection.* `(From architecture.md Sections 7, 8)`
2.  **Storage:** Articles and scores are stored in SQLite via `internal/db`.
    *   *Debugging/Improvements: Check DB health/schema; check `llm_scores` consistency; optimize indexes; consider score versioning/pruning; evaluate DB scaling.* `(From architecture.md Section 5)`
3.  **Analysis Trigger:** Analysis is triggered (e.g., via API call to `cmd/server`/`internal/api` or `cmd/score_articles`).
4.  **LLM Interaction:** `internal/llm` manages calls to external LLMs using `internal/llm/service_http.go`. Ensemble methods (`internal/llm/ensemble.go`) use multiple models/prompts defined in `configs/composite_score_config.json`.
    *   *Debugging/Improvements: Check job logs (`cmd/score_articles`); verify LLM API keys/quota; examine `llm_scores.metadata`; add robust error handling/retries; make models/prompts configurable; monitor costs; improve response parsing.* `(From architecture.md Section 6)`
5.  **Score Calculation & Persistence:** `internal/llm/composite_score_fix.go` and `internal/llm/score_calculator.go` calculate a composite score based on `configs/composite_score_config.json`. The `internal/llm/score_manager.go` orchestrates this, persists the final score via `internal/db`, invalidates caches, and updates progress. Caching of individual LLM responses is handled by `internal/llm/cache.go`.
    *   *Debugging/Improvements: Verify input scores/logic; check handling of missing/invalid scores (e.g., scores are ignored if `handle_invalid` is "ignore", or replaced by `default_missing` if "default"); watch for NaN/Inf; explore alternative algorithms; make calculation configurable; add confidence metric.* `(From architecture.md Section 4)`
6.  **Presentation:** `cmd/server/main.go` runs the web server. `internal/api` exposes data via REST endpoints (e.g., `/articles`, `/articles/{id}/bias`) and potentially serves a web UI (`web/`). `internal/llm/progress_manager.go` tracks async task status for endpoints like `/llm/score-progress/:id`.
    *   *Debugging/Improvements (API): Check API logs; verify JSON structure; monitor DB query performance; add API caching; strengthen input validation; standardize errors; tune pagination.* `(From architecture.md Section 3)`
    *   *Debugging/Improvements (Frontend): Check JS console/network tab; verify CSS/DOM rendering; implement caching; add loading indicators/error messages; add score confidence indicator.* `(From architecture.md Sections 1, 2)`

---

## 2. Configuration Files

Configuration is crucial for adapting the application's behavior without code changes. Key files include:

*   **`.env` (Project Root):**
    *   **Purpose:** Stores environment variables, primarily secrets and external service configurations. Loaded by `godotenv`.
    *   **Expected Variables:**
        *   `LLM_API_KEY`: Primary API key for the LLM service (e.g., OpenRouter).
        *   `LLM_API_KEY_SECONDARY`: (Optional) Backup API key for rate limit fallback.
        *   `LLM_BASE_URL`: Base URL for the LLM service (defaults to OpenRouter if not set).
        *   `DATABASE_URL`: (Potentially, although `news.db` seems hardcoded often) Connection string or path for the database.
*   **`configs/feed_sources.json` (Project Root `/configs`):**
    *   **Purpose:** Defines the list of RSS feed URLs to be fetched by the `internal/rss.Collector` when run by the main server (`cmd/server/main.go`).
    *   **Expected Structure:** A JSON array of strings, where each string is a valid RSS/Atom feed URL.
        ```json
        [
          "http://rss.cnn.com/rss/cnn_topstories.rss",
          "http://feeds.foxnews.com/foxnews/latest",
          "..."
        ]
        ```
*   **`configs/composite_score_config.json` (Project Root `/configs`):**
    *   **Purpose:** Defines the strategy for LLM ensemble analysis and composite score calculation. Loaded by `internal/llm.LoadCompositeScoreConfig`.
    *   **Expected Structure:** A JSON object defining models, scoring parameters, and calculation methods.
        ```json
        {
          "models": [
            {
              "name": "ModelName1 (e.g., gpt-3.5-turbo)",
              "url": "API endpoint if different from base URL",
              "perspective": "left | center | right | other", // Used for grouping/weighting
              "role": "Optional descriptive role"
            },
            { ... }
          ],
          "min_score": -1.0, // Minimum acceptable score
          "max_score": 1.0,  // Maximum acceptable score
          "default_missing": 0.0, // Value to use if a perspective is missing (used if handle_invalid is "default")
          "handle_invalid": "ignore | default", // How to treat NaN/Inf scores
          "formula": "average | weighted", // How to combine perspective scores
          "weights": { // Used if formula is "weighted"
            "left": 0.33,
            "center": 0.34,
            "right": 0.33
          },
          "confidence_method": "average | min | max | spread_based", // How to calculate overall confidence
          "confidence_params": { // Parameters specific to the confidence_method
            "min_count": 2 // e.g., Min perspectives needed for spread_based
          }
        }
        ```
*   **`internal/llm/configs/*.txt` (Internal LLM Configs):**
    *   **Purpose:** Contain text templates for different prompt variants used in `internal/llm/ensemble.go`. Loaded by `loadPromptVariants`.
    *   **Files:** `DefaultPromptVariant.txt`, `concise_json_prompt.txt`, etc.

---

## 3. Core Workflow Components

### 3.1. Main Application Server (`cmd/server/main.go`)

*   **Purpose:** This is the **primary entry point** for the main application. It sets up and runs a Gin web server that serves both the web interface and the backend API.
*   **Key Components:**
    *   **Initialization (`initServices`):** Loads environment variables (`.env`) and initializes core services:
        *   Database connection (`db.InitDB`).
        *   LLM client (`llm.NewLLMClient`).
        *   RSS feed collector (`rss.NewCollector`) using `configs/feed_sources.json`.
        *   Score manager (`llm.NewScoreManager`).
    *   **Gin Router Setup:**
        *   Loads HTML templates (`web/*.html`).
        *   Serves static files (`./web`).
        *   Defines API routes by calling `api.RegisterRoutes`.
        *   Defines web interface routes (e.g., `/`, `/articles`, `/article/:id`) and metrics endpoints (`/metrics/*`).
        *   Sets up Swagger UI (`/swagger/*any`).
*   **Dependencies:** Gin, SQLx, godotenv, Swaggo, and most `internal/` packages (`db`, `llm`, `rss`, `api`, `metrics`).
*   **Usage:** `go run cmd/server/main.go` - starts the server (default port 8080).

### 3.2. API Layer (`internal/api/`)

This package handles incoming HTTP requests, routes them to appropriate logic, interacts with backend services (LLM, DB, RSS), and formats responses.

*   **`api.go`:**
    *   **Purpose:** Defines the API structure, routes using Gin, and implements many request handlers. Orchestrates interactions between HTTP requests and backend logic.
    *   **`RegisterRoutes`:** Central function defining all `/api/*` endpoints (articles, feeds, scoring, analysis, feedback) and mapping them to handlers. Injects dependencies (DB, RSS Collector, LLM Client, Score Manager) into handlers.
    *   **Middleware:** Uses `SafeHandler` for panic recovery.
    *   **Handlers:** Implements logic for endpoints like `getArticlesHandler`, `reanalyzeHandler` (triggers async scoring via `ScoreManager` and `ProgressManager`), `biasHandler`, `ensembleDetailsHandler`, `feedbackHandler`, `scoreProgressSSEHandler` (uses `ProgressManager` for SSE).
    *   **Caching:** Uses `articlesCache` (instance of `SimpleCache`) for `/api/articles`.
    *   **Progress Tracking:** Manages a `progressMap` for async scoring tasks.
*   **`models.go`:**
    *   **Purpose:** Defines Data Transfer Objects (DTOs) for API request payloads (e.g., `CreateArticleRequest`, `FeedbackRequest`, `ManualScoreRequest`) and response bodies (e.g., `Article`, `ScoreResponse`, `StandardResponse`, `ErrorResponse`). Distinct from `internal/db` models.
    *   **Features:** Uses `json` tags for serialization, `binding` tags for validation, and Swagger annotations.
*   **`response.go`:**
    *   **Purpose:** Provides utility functions for standardized API responses and structured logging.
    *   **Key Functions:** `RespondSuccess` (returns `StandardResponse`), `RespondError` (maps `apperrors.AppError` to HTTP status and `ErrorResponse`), `LogError`, `LogPerformance`.
*   **`cache.go`:**
    *   **Purpose:** Defines a simple, thread-safe, in-memory cache (`SimpleCache`) with Time-To-Live (TTL) support.
    *   **Usage:** Used by `api.go` for the `/api/articles` endpoint.
*   **`errors.go`:**
    *   **Purpose:** Defines constants for API error codes (`ErrValidation`, `ErrNotFound`, etc.) and pre-instantiated `apperrors.AppError` variables for common API errors.
    *   **Role:** Standardizes API error representation.
*   **Debugging Points:**
    *   Check API logs for errors (especially 5xx responses).
    *   Verify the structure of the JSON response matches API models and client expectations.
    *   Ensure correct article IDs or other parameters are used in requests.
    *   Monitor database query performance within handlers (e.g., using logs or DB profiling).

*(Note: `handlers.go` and `db_operations.go` in this package contain minimal or commented-out code).*

### 3.3. LLM Analysis Core (`internal/llm/`)

This package contains the core logic for interacting with LLMs, calculating bias scores, and managing the analysis process.

*   **`llm.go`:**
    *   **Purpose:** Defines the central `LLMClient`, orchestrating LLM interactions, caching, configuration, and analysis workflows.
    *   **`LLMClient`:** Holds dependencies: `LLMService` interface, `Cache`, `*sqlx.DB`, `CompositeScoreConfig`.
    *   **Core Functions:** `NewLLMClient` (initialization), `AnalyzeContent` / `ScoreWithModel` (single model analysis with caching), utilizes `EnsembleAnalyze`. Manages DB interactions for scores/articles.
*   **`ensemble.go`:**
    *   **Purpose:** Implements the `EnsembleAnalyze` method for `LLMClient`, orchestrating multi-model and multi-prompt analysis for robustness.
    *   **`EnsembleAnalyze`:** Iterates through models (from `CompositeScoreConfig`) and prompt variants (`loadPromptVariants`), calls `callLLM` for each, aggregates results per model, computes the final score using `ComputeCompositeScoreWithConfidenceFixed`, and packages detailed metadata.
    *   **`callLLM`:** Helper for single model/prompt call with retries and embedded error checking.
    *   **`parseNestedLLMJSONResponse`:** Robust parser for LLM responses (JSON, Markdown, Regex fallback).
    *   **`loadPromptVariants`:** Provides different prompt structures (from `internal/llm/configs/*.txt`).
*   **`service_http.go`:**
    *   **Purpose:** Implements the `LLMService` interface via `HTTPLLMService` for actual HTTP calls to external LLM APIs (e.g., OpenRouter).
    *   **`HTTPLLMService`:** Holds Resty client, API keys (primary/backup), base URL.
    *   **`ScoreContent`:** Orchestrates the API call, including sophisticated retry logic for rate limits (using backup key, trying alternative models from config). Calls `callLLMAPIWithKey`.
    *   **`callLLMAPIWithKey`:** Executes the HTTP POST request.
*   **`composite_score_fix.go`:**
    *   **Purpose:** Contains the primary logic (`ComputeCompositeScoreWithConfidenceFixed`) for calculating the final composite score and confidence from multiple individual `db.LLMScore` inputs, based on `CompositeScoreConfig`.
    *   **Key Helpers:** `MapModelToPerspective`, `checkForAllZeroResponses`, `mapModelsToPerspectives`, `processScoresByPerspective`, `calculateCompositeScore`, `calculateConfidence`.
    *   **Role:** Core calculation engine, highly configurable via `CompositeScoreConfig`.
*   **`score_calculator.go`:**
    *   **Purpose:** Defines the `ScoreCalculator` interface and `DefaultScoreCalculator` implementation.
    *   **`DefaultScoreCalculator`:** Calculates an average score/confidence across perspectives ("left", "center", "right") after mapping models using `getPerspective` and extracting confidence via `extractConfidence`. Used by `ScoreManager`.
*   **`score_manager.go`:**
    *   **Purpose:** Defines `ScoreManager` to orchestrate the *final* stages of scoring: calculating the composite score (via `ScoreCalculator`), persisting it (`db.UpdateArticleScoreLLM`), invalidating caches (`InvalidateScoreCache`), and updating progress (`ProgressManager`).
    *   **`UpdateArticleScore`:** Primary method, called after individual LLM analyses are complete.
    *   **Dependencies:** DB, Cache, ScoreCalculator, ProgressManager.
*   **`cache.go`:**
    *   **Purpose:** Provides `Cache` (`sync.Map`-based) for storing/retrieving `db.LLMScore` JSON, keyed by content hash and model name.
    *   **Usage:** Used by `LLMClient` to avoid redundant API calls and by `ScoreManager` for invalidation.
*   **`progress_manager.go`:**
    *   **Purpose:** Defines `ProgressManager` for in-memory tracking of asynchronous article scoring tasks (status, percentage, errors). Used by `ScoreManager` and exposed via API (e.g., SSE endpoint).
    *   **Features:** Thread-safe map (`progressMap`), automatic cleanup routine for completed/stale entries.
*   **`composite_score_utils.go`:**
    *   **Purpose:** Utilities for loading `CompositeScoreConfig` from `configs/composite_score_config.json` (robust path finding) and numerical helpers (`minNonNil`, `maxNonNil`, `scoreSpread`).
    *   **`LoadCompositeScoreConfig`:** Reads and parses the config file *on each call* (no caching).
*   **`errors.go`:**
    *   **Purpose:** Defines package-level error variables for common LLM issues (`ErrBothLLMKeysRateLimited`, `ErrLLMServiceUnavailable`, `ErrAllPerspectivesInvalid`).
*   **Debugging Points:**
    *   Check job logs (`cmd/score_articles`) for errors during batch processing.
    *   Verify LLM API key validity and quota usage.
    *   Examine the `metadata` column in `llm_scores` for raw LLM response details/errors.
    *   Step through score calculation logic (`composite_score_fix.go`, `score_calculator.go`) for correctness.
    *   Verify input scores passed to calculation functions.
    *   Check if handling of missing/invalid scores (based on `handle_invalid` config) is appropriate.
    *   Watch for NaN or Infinity results in edge cases.

*(Note: `internal/llm/configs/` contains JSON/text configuration files used by this package, see Section 2).*

### 3.4. Database Layer (`internal/db/`)

*   **Purpose:** Acts as the data access layer (DAL) / persistence layer for the application using SQLite. Defines data models and provides functions for all database interactions.
*   **`db.go`:**
    *   **Data Models:** Defines core structs mapped to DB tables (`Article`, `LLMScore`, `Feedback`, `Label`) with `db` and `json` tags.
    *   **Initialization (`InitDB`):** Opens connection and calls `createSchema`.
    *   **Schema Management (`createSchema`):** Executes `CREATE TABLE IF NOT EXISTS` for all tables, defining columns, primary keys, foreign keys, UNIQUE constraints, and indexes.
    *   **CRUD Operations:** Provides functions like `InsertArticle` (with `ON CONFLICT DO NOTHING`), `InsertLLMScore`, `InsertFeedback`, `InsertLabel`, `FetchArticles` (with filtering/pagination), `FetchArticleByID`, `FetchLLMScores`, `UpdateArticleScoreLLM`, `ArticleExistsByURL`, `ArticleExistsBySimilarTitle`.
    *   **Error Handling:** Uses `handleError` to wrap DB errors into `apperrors`.
    *   **Dependencies:** `modernc.org/sqlite`, `github.com/jmoiron/sqlx`, `internal/apperrors`.
*   **Debugging Points:**
    *   Check database connection health and file permissions.
    *   Verify the table schemas (`articles`, `llm_scores`, `feedback`, `labels`) match the `createSchema` definitions.
    *   Look for missing scores in `llm_scores` for specific `article_id` values.
    *   Assess the performance of indexes, especially on `articles(url)` and `llm_scores(article_id)`.
    *   Check for SQLite locking issues (`SQLITE_BUSY`) if multiple processes write concurrently.

### 3.5. Data Ingestion (`internal/rss/`)

*   **Purpose:** Responsible for fetching, parsing, validating, deduplicating, and storing articles from RSS feeds.
*   **`rss.go`:**
    *   **`Collector` Struct:** Holds DB connection, feed URLs, cron scheduler (`github.com/robfig/cron/v3`), and LLMClient (though use isn't apparent in main fetch flow).
    *   **Core Functions:**
        *   `NewCollector`: Initialization.
        *   `StartScheduler`: Configures cron job (`@every 30m`) to run `FetchAndStore`.
        *   `FetchAndStore`: Iterates feeds, calls `fetchFeed` (uses `gofeed` parser), then `processFeedItem`.
        *   `processFeedItem`: Handles validation (`isValidItem`), deduplication (`db.ArticleExistsByURL`, `db.ArticleExistsBySimilarTitle`), content extraction (`extractContent`), and storage (`storeArticle` -> `db.InsertArticle`).
        *   `CheckFeedHealth`: Checks accessibility of feed URLs.
    *   **Dependencies:** `gofeed`, `cron`, `sqlx`, `internal/db`, `internal/llm`.
*   **Debugging Points:**
    *   Check job logs (`cmd/fetch_articles` or server logs if run via scheduler) for errors.
    *   Verify that RSS feed URLs in the configuration (`configs/feed_sources.json`) are valid and active.
    *   Check for network connectivity issues when reaching feeds.
    *   Examine parsing logic (`gofeed`) for compatibility with specific feed formats; look for skipped items.
    *   Ensure database write permissions and check for DB write errors during `InsertArticle`.
    *   Examine the logic for handling duplicate articles (URL or similar title).

---

## 4. Core Supporting Packages

### 4.1. Shared Data Models (`internal/models/`)

*   **Purpose:** Defines shared data structures used across different packages, often for representing state or specific concepts not tied directly to DB schema or API contracts.
*   **`progress.go`:**
    *   **`ProgressState` Struct:** Represents the status of a long-running task (like scoring). Includes `Step`, `Message`, `Percent`, `Status`, `Error`, `FinalScore`, `LastUpdated`. Used by `ProgressManager` and likely returned in API responses.

### 4.2. Application Errors (`internal/apperrors/`)

*   **Purpose:** Provides a standardized way to represent and handle errors throughout the application.
*   **`errors.go`:**
    *   **`AppError` Struct:** Custom error type with `Code`, `Message`, and `Cause` (for wrapping).
    *   **Key Functions:** `New`, `Wrap`, `HandleError`, `Join`. Implements `error`, `errors.Is`, `errors.Unwrap` interfaces.
    *   **Role:** Enables consistent error logging, checking, and mapping to API responses.

### 4.3. Metrics (`internal/metrics/`)

*   **Purpose:** Defines structures and functions related to application metrics, both for Prometheus scraping and direct database querying.
*   **`metrics.go`:**
    *   **Purpose:** Defines structs for aggregated DB metrics (e.g., `ValidationMetric`, `FeedbackSummary`, `UncertaintyRate`) and functions to query corresponding DB tables/views (`GetValidationMetrics`, etc.).
    *   **Usage:** Likely used by `/metrics/*` API endpoints. Assumes background population of metrics tables.
*   **`prom.go`:**
    *   **Purpose:** Sets up Prometheus metrics (Counters, Gauges) for LLM API interactions (`LLMRequestsTotal`, `LLMFailuresTotal`, `LLMFailureStreak`) using `prometheus/client_golang`.
    *   **Usage:** Provides functions (`IncLLMRequest`, `IncLLMFailure`, etc.) to be called from the LLM interaction code. Requires `InitLLMMetrics` call at startup and a Prometheus scrape endpoint.

---

## 5. Command-Line Tools (`cmd/`)

These are standalone executables, typically for utility, testing, or batch processing tasks.

### 5.1. Data Management & Migration

*   **`cmd/import_labels/main.go`:** Imports labeled data (CSV/JSON) into the `labels` table for validation/training.
*   **`cmd/clear_articles/main.go`:** Destructive tool to DELETE all records from `articles` and `llm_scores` tables.
*   **`cmd/delete_mock_scores/main.go`:** Deletes specific scores from `llm_scores` (hardcoded `article_id=133`).
*   **`cmd/migrate_feedback_schema/main.go`:** Idempotent script to add `category`, `ensemble_output_id`, `source` columns to the `feedback` table if they don't exist.

### 5.2. Batch Processing & Fetching

*   **`cmd/score_articles/main.go`:** Fetches articles from DB, uses `LLMClient` (and worker pool) to analyze/score them, and inserts scores into `llm_scores`. Core batch analysis tool.
*   **`cmd/fetch_articles/main.go`:** Manually triggers article fetching using `rss.Collector.FetchAndStore`. Uses a *hardcoded* list of feed URLs (potentially different from the server's config).

### 5.3. Validation & Reporting

*   **`cmd/validate_labels/main.go`:** Evaluates LLM performance against data in the `labels` table. Fetches labels, runs `EnsembleAnalyze`, compares results, calculates metrics (Accuracy, Precision, Recall, F1, Confusion Matrix), and saves results/flagged cases to JSON files.
*   **`cmd/generate_report/main.go`:** Fetches data from the *running server's* `/metrics/*` API endpoints and saves them to timestamped CSV files. Includes basic alerting for high uncertainty.

### 5.4. Querying & Testing

*   **`cmd/query_article/main.go`:** Retrieves and displays a single article from the DB by ID.
*   **`cmd/query_scores/main.go`:** Retrieves and displays all `llm_scores` for a hardcoded article ID (681).
*   **`cmd/test_llm/main.go`:** Simple test client to send a predefined article/prompt to the configured LLM service (using `llm.HTTPLLMService`).
*   **`cmd/test_parse/main.go`:** Tests parsing a local `sample_feed.xml` file using `gofeed`.

---

## 6. Mocking & Testing Utilities

*   **`mock_llm_service.go` (Root Directory):**
    *   **Purpose:** Simulates an LLM service for testing. Listens on a port and returns a predefined score/label for `/analyze` requests. Useful for testing components that depend on an LLM service without making actual API calls.
*   **`internal/testing/coordinator.go`:**
    *   **Purpose:** Defines a `TestCoordinator` framework to run multiple test suites defined as external commands (Go tests, scripts, etc.), potentially in parallel, capturing output and providing a basic pass/fail report. Useful for integration testing.

---

## 7. Final Remarks

This document provides a logically structured overview of the NewsBalancer Go codebase, focusing on core components and their interactions. It covers the main server, API layer, LLM analysis core, database interactions, RSS ingestion, supporting packages, configuration files, command-line utilities, and testing tools. This structure should serve as a useful reference for understanding the system's architecture and data flow, facilitating future development and maintenance.

## Database Schema

The application uses SQLite as its database. The schema is defined in `internal/db/db.go` and includes the following tables:

- **articles**: Stores article information and metadata, including composite scores.
- **llm_scores**: Stores individual LLM model scores for articles.
  - Contains a `UNIQUE(article_id, model)` constraint to enable `ON CONFLICT` clauses in SQL queries.
  - This constraint is critical for the proper functioning of ensemble score updates.
- **feedback**: Stores user feedback on articles.
- **labels**: Stores training labels for the system.

## OpenRouter Error Handling

The application includes robust error handling for the OpenRouter LLM service, which is used for article analysis. This section explains how different types of OpenRouter errors are handled.

### OpenRouter Error Types

| HTTP Status | Error Type | Description | Retry Strategy |
|-------------|------------|-------------|----------------|
| 429 | Rate Limit | Occurs when exceeding 1 request per credit per second | Respects Retry-After header, falls back to secondary key |
| 402 | Credits Exhausted | Account has negative credit balance | No automatic retry, requires account top-up |
| 401 | Authentication | Invalid API key | No retry, requires API key verification |
| 4xx/5xx | Other Errors | Bad request, server errors, etc. | Limited retries with exponential backoff |

### Error Response Format

When an OpenRouter error occurs, the API responds with:

```json
{
  "code": "llm_error_type",
  "message": "Human-readable error description",
  "details": {
    "llm_status_code": 429,
    "llm_message": "Original error message from OpenRouter",
    "llm_error_type": "rate_limit",
    "retry_after": 30
  }
}
```

### Monitoring and Metrics

OpenRouter errors are tracked using Prometheus metrics:
- `llm_requests_total`: Total number of requests made to OpenRouter
- `llm_failures_total`: Total number of failed requests to OpenRouter
- `llm_rate_limit_total`: Number of rate limit errors
- `llm_auth_failure_total`: Number of authentication failures
- `llm_credits_exhausted_total`: Number of credit exhaustion errors
- `llm_streaming_errors_total`: Number of streaming-related errors

### Troubleshooting OpenRouter Errors

1. **Rate Limit Errors**:
   - Check for multiple concurrent requests
   - Verify secondary API key is configured
   - Consider implementing request throttling

2. **Authentication Errors**:
   - Verify API key in `.env` file
   - Check for proper API key format
   - Confirm account is active on OpenRouter

3. **Credits Exhausted**:
   - Top up your OpenRouter account
   - Monitor usage patterns to avoid unexpected exhaustion
   - Consider implementing usage alerts

4. **Streaming Errors**:
   - Check for network connectivity issues
   - Verify OpenRouter streaming endpoint status
   - Consider falling back to non-streaming API