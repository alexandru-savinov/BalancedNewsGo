# Codebase Documentation: NewsBalancer

## 1. Introduction

This document provides a high-level overview and detailed documentation of the Go codebase for the NewsBalancer project. The primary goal of NewsBalancer is to fetch news articles from various RSS feeds, analyze their political bias using Large Language Models (LLMs), store the results, and expose this data through a web interface and API.

**Latest Test Status:**
- Most Go unit, integration, and end-to-end tests pass. Notably, `internal/llm` unit tests have some outstanding failures (see `docs/testing.md`).
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
2.  **Storage:** Articles and scores are stored in SQLite (typically `news.db`) via `internal/db`.
    *   *Debugging/Improvements: Check DB health/schema; check `llm_scores` consistency (especially the `UNIQUE(article_id, model)` constraint); optimize indexes; consider score versioning/pruning; evaluate DB scaling.* `(From architecture.md Section 5)`
3.  **Analysis Trigger:** Analysis is triggered (e.g., via API call to `cmd/server`/`internal/api` or `cmd/score_articles`).
4.  **LLM Interaction:** `internal/llm` manages calls to external LLMs using `internal/llm/service_http.go`. Ensemble methods (`internal/llm/ensemble.go`) use multiple models/prompts defined in `configs/composite_score_config.json`. The system **averages duplicate scores and confidences** if multiple results are found for the same model/perspective during an analysis pass.
    *   *Debugging/Improvements: Check job logs (`cmd/score_articles`); verify LLM API keys/quota; examine `llm_scores.metadata`; add robust error handling/retries; make models/prompts configurable; monitor costs; improve response parsing.* `(From architecture.md Section 6)`
5.  **Score Calculation & Persistence:** `internal/llm/composite_score_fix.go` contains the primary logic for calculating the final composite score and confidence from multiple individual `db.LLMScore` inputs, driven by `configs/composite_score_config.json`. `internal/llm/score_calculator.go` provides the interface and default implementation. The `internal/llm/score_manager.go` orchestrates this, persists the final score via `internal/db`, invalidates caches (`internal/llm/cache.go` for LLM responses, `internal/api/cache.go` for API responses), and updates progress.
    *   *Debugging/Improvements: Verify input scores/logic; check handling of missing/invalid scores (e.g., scores are ignored if `handle_invalid` is "ignore", or replaced by `default_missing` if "default"); watch for NaN/Inf; explore alternative algorithms; make calculation configurable; add confidence metric.* `(From architecture.md Section 4)`
6.  **Presentation:** `cmd/server/main.go` runs the web server. `internal/api` exposes data via REST endpoints (e.g., `/articles`, `/articles/{id}/bias`) and serves a modern client-side web UI (`web/`). `internal/llm/progress_manager.go` (used by `internal/api/api.go` and `internal/llm/score_manager.go`) tracks asynchronous task status for endpoints like `/api/llm/score-progress/:id` (SSE).
    *   *Debugging/Improvements (API): Check API logs; verify JSON structure; monitor DB query performance; add API caching; strengthen input validation; standardize errors; tune pagination.* `(From architecture.md Section 3)`
    *   *Debugging/Improvements (Frontend): Check JS console/network tab; verify CSS/DOM rendering; implement caching; add loading indicators/error messages; add score confidence indicator.* `(From architecture.md Sections 1, 2)`

---

## 2. Configuration Files

Configuration is crucial for adapting the application's behavior without code changes. Key files include:

*   **`.env.example` (Project Root):**
    *   **Purpose:** Example environment configuration. Copy this file to `.env` and populate your secrets. The application loads variables from `.env` via `godotenv`.
    *   **Expected Variables:**
        *   `LLM_API_KEY`: Primary API key for the LLM service (e.g., OpenRouter).
        *   `LLM_API_KEY_SECONDARY`: (Optional) Backup API key for rate limit fallback.
        *   `LLM_BASE_URL`: Base URL for the LLM service (defaults to OpenRouter if not set).
        *   `DATABASE_URL`: This variable is listed as a possibility, but the main server (`cmd/server/main.go`) and most utility tools in `cmd/` currently default to using a SQLite database file named `news.db` located in the execution directory. Some tools might offer flags to specify a different DB path (e.g., `cmd/import_labels/main.go`).
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
    *   **Initialization (`initServices`):** Loads environment variables from `.env` (copy `.env.example` and fill it in) and initializes core services:
        *   Database connection (`db.InitDB`).
        *   LLM client (`llm.NewLLMClient`).
        *   RSS feed collector (`rss.NewCollector`) using `configs/feed_sources.json`.
        *   Score manager (`llm.NewScoreManager`).
    *   **Gin Router Setup:**
        *   Loads HTML templates (`web/*.html`).
        *   Serves static files (`./web`).
        *   Defines API routes by calling `api.RegisterRoutes`.
        *   **Web Interface Options:**
            *   **Modern (Default):** Serves static HTML files (`index.html`, `article.html`) that use client-side JavaScript to consume API endpoints.
            *   **Legacy (Optional):** A server-side rendering mode enabled via the `--legacy-html` flag or `LEGACY_HTML=true` environment variable. This uses the `legacyArticlesHandler` and `legacyArticleDetailHandler` functions.
        *   Defines metrics endpoints (`/metrics/*`).
        *   Sets up Swagger UI (`/swagger/*any`).
*   **Dependencies:** Gin, SQLx, godotenv, Swaggo, and most `internal/` packages (`db`, `llm`, `rss`, `api`, `metrics`).
*   **Usage:** `go run cmd/server/main.go` - starts the server (default port 8080).
    *   *Note: If running the server directly or via `make run`, ensure port 8080 is free. Port conflict errors (e.g., "Only one usage of each socket address") can occur if the port is already in use. Refer to `docs/testing.md` for troubleshooting port conflicts.*
*   **Note:** Several obsolete functions exist in the codebase related to the legacy web rendering:
    *   `articleDetailHandler`: A placeholder function marked with a TODO that was never fully implemented
    *   `articlesHandler`: Unused duplicate of the legacy handler functionality
    *   A commented-out background reprocessing loop that was disabled for debugging

### 3.2. API Layer (`internal/api/`)

This package handles incoming HTTP requests, routes them to appropriate logic, interacts with backend services (LLM, DB, RSS), and formats responses.

*   **`api.go`:**
    *   **Purpose:** Defines the API structure, routes using Gin, and implements many request handlers. Orchestrates interactions between HTTP requests and backend logic. It is the primary file for the API layer.
    *   **`RegisterRoutes`:** Central function within `internal/api/api.go` defining all `/api/*` endpoints (articles, feeds, scoring, analysis, feedback) and mapping them to handlers. Injects dependencies (DB, RSS Collector, LLM Client, Score Manager, ProgressManager, APICache) into handlers.
    *   **Middleware:** Uses `SafeHandler` for panic recovery.
    *   **Handlers:** Implements logic for endpoints like `getArticlesHandler`, `reanalyzeHandler` (triggers async scoring via `ScoreManager` and `ProgressManager`), `biasHandler`, `ensembleDetailsHandler`, `feedbackHandler`, `scoreProgressSSEHandler` (uses `ProgressManager` for Server-Sent Events on `/api/llm/score-progress/:id`).
    *   **Caching:** Uses `articlesCache` (instance of `SimpleCache` from `internal/api/cache.go`) for `/api/articles`.
    *   **Progress Tracking:** The `ProgressManager` (from `internal/llm/progress_manager.go`) is injected and used by handlers (e.g., for re-analysis tasks) to track and expose progress, notably via the SSE endpoint.
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

### 3.3. Web Interface (`web/`)

The web interface provides a user-friendly way to interact with the NewsBalancer system. It consists of HTML templates and client-side JavaScript that interact with the backend API.

*   **Implementation Approaches:**
    *   **Modern (Default):** Client-side rendering using static HTML files and JavaScript. This is the primary implementation.
    *   **Legacy:** Server-side rendering (enabled via the `--legacy-html` flag), which is maintained for backward compatibility but is no longer the recommended approach.

*   **HTML Templates:**
    *   **`index.html`:** The main page that displays a list of articles with their political bias scores. Includes filtering, sorting, and pagination controls.
    *   **`article.html`:** Detailed view of a single article, showing its full content, bias analysis, ensemble details, and a feedback form.
*   **JavaScript:**
    *   **`web/js/list.js`:** 
        *   **Purpose:** Handles the article list page functionality.
        *   **Key Features:** Client-side caching, pagination, filtering (by source, leaning, confidence), sorting, and dynamic rendering of article cards.
        *   **API Integration:** Fetches article data from `/api/articles` endpoint with pagination and filter parameters.
    *   **`web/js/article.js`:** 
        *   **Purpose:** Manages the article detail page.
        *   **Key Features:** Displays article content, bias visualization, confidence indicators, ensemble details showing individual model scores, and user feedback submission.
        *   **API Integration:** Fetches article data from `/api/articles/:id`, ensemble details from `/api/articles/:id/ensemble-details`, and submits feedback to `/api/feedback`.
*   **Common Features:**
    *   **Client-side Caching:** Both scripts implement a caching mechanism with expiry to reduce redundant API calls.
    *   **Error Handling:** Comprehensive error handling with user-friendly messages.
    *   **Loading States:** Visual indicators during data fetching operations.
    *   **Responsive Design:** CSS styles adapt to different screen sizes.
    *   **Visualization:** Bias slider with visual indicators for political leaning and confidence levels.
*   **Debugging Points:**
    *   Check browser console for JavaScript errors.
    *   Verify network requests in browser developer tools to ensure proper API interaction.
    *   Test caching functionality by refreshing the page and observing reduced API calls.
    *   Validate proper rendering of bias indicators and confidence visualizations.
    *   Ensure feedback submission is working correctly.

### 3.4. LLM Analysis Core (`internal/llm/`)

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
    *   **Purpose:** Contains the primary logic (`ComputeCompositeScoreWithConfidenceFixed`) for calculating the final composite score and confidence from multiple individual `db.LLMScore` inputs. This calculation is highly configurable via the parameters in `configs/composite_score_config.json` (defining models, perspectives, formula, weights, confidence method, etc.).
    *   **Key Helpers:** `MapModelToPerspective`, `checkForAllZeroResponses`, `mapModelsToPerspectives`, `processScoresByPerspective`, `calculateCompositeScore`, `calculateConfidence`.
    *   **Role:** Core calculation engine for ensemble scoring.
*   **`score_calculator.go`:**
    *   **Purpose:** Defines the `ScoreCalculator` interface and `DefaultScoreCalculator` implementation.
    *   **`DefaultScoreCalculator`:** Calculates an average score/confidence across perspectives ("left", "center", "right") after mapping models using `getPerspective` and extracting confidence via `extractConfidence`. Used by `ScoreManager`.
*   **`score_manager.go`:**
    *   **Purpose:** Defines `ScoreManager` to orchestrate the *final* stages of scoring: calculating the composite score (via `ScoreCalculator`), persisting it (`db.UpdateArticleScoreLLM`), invalidating caches (`InvalidateScoreCache`), and updating progress (`ProgressManager`).
    *   **`UpdateArticleScore`:** Primary method, called after individual LLM analyses are complete.
    *   **Dependencies:** DB, Cache, ScoreCalculator, ProgressManager.
*   **`cache.go` (`internal/llm/cache.go`):**
    *   **Purpose:** Provides `Cache` (`sync.Map`-based) for storing/retrieving individual `db.LLMScore` JSON responses from the LLM service, keyed by content hash and model name.
    *   **Usage:** Used by `LLMClient` to avoid redundant API calls to external LLM services. Invalidated by `ScoreManager` when scores are updated.
*   **`progress_manager.go` (`internal/llm/progress_manager.go`):**
    *   **Purpose:** Defines `ProgressManager` for in-memory tracking of asynchronous article scoring tasks (status, percentage, errors).
    *   **Usage:** Used by `ScoreManager` to update progress during analysis and by `internal/api/api.go` to serve progress information, notably via the Server-Sent Events (SSE) endpoint `/api/llm/score-progress/:id`.
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

### 3.5. Database Layer (`internal/db/`)

*   **Purpose:** Acts as the data access layer (DAL) / persistence layer for the application using SQLite. Defines data models and provides functions for all database interactions. The schema is defined within this package.
*   **`db.go`:**
    *   **Data Models:** Defines core structs mapped to DB tables (`Article`, `LLMScore`, `Feedback`, `Label`) with `db` and `json` tags.
    *   **Initialization (`InitDB`):** Opens connection to a specified SQLite database file (typically `news.db` for the main server and tools) and calls `createSchema`.
    *   **Schema Management (`createSchema`):** Executes `CREATE TABLE IF NOT EXISTS` for all tables: `articles`, `llm_scores`, `feedback`, `labels`. Defines columns, primary keys, foreign keys, and crucially, a `UNIQUE(article_id, model)` constraint on the `llm_scores` table to enable `ON CONFLICT` clauses for upserting scores. Also creates relevant indexes.
    *   **CRUD Operations:** Provides functions like `InsertArticle` (with `ON CONFLICT DO NOTHING`), `InsertLLMScore` (uses `ON CONFLICT` to update), `InsertFeedback`, `InsertLabel`, `FetchArticles` (with filtering/pagination), `FetchArticleByID`, `FetchLLMScores`, `UpdateArticleScoreLLM`, `ArticleExistsByURL`, `ArticleExistsBySimilarTitle`.
    *   **Error Handling:** Uses `handleError` to wrap DB errors into `apperrors`.
    *   **Dependencies:** `modernc.org/sqlite`, `github.com/jmoiron/sqlx`, `internal/apperrors`.
*   **Debugging Points:**
    *   Check database connection health and file permissions.
    *   Verify the table schemas (`articles`, `llm_scores`, `feedback`, `labels`) match the `createSchema` definitions.
    *   Look for missing scores in `llm_scores` for specific `article_id` values.
    *   Assess the performance of indexes, especially on `articles(url)` and `llm_scores(article_id)`.
    *   Check for SQLite locking issues (`SQLITE_BUSY`) if multiple processes write concurrently.

### 3.6. Data Ingestion (`internal/rss/`)

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

*   **Purpose:** Provides a standardized way to represent and handle errors throughout the application, ensuring consistent error structures for logging and API responses.
*   **`errors.go`:**
    *   **`AppError` Struct:** Custom error type with `Code` (e.g., "not_found", "validation_error"), `Message` (user-friendly), and `Cause` (for wrapping underlying errors).
    *   **Key Functions:** `New` (creates a new AppError), `Wrap` (wraps an existing error with AppError context), `HandleError` (often used in `internal/db` to convert DB errors to AppErrors), `Join` (combines multiple errors). Implements `error`, `errors.Is`, `errors.Unwrap` interfaces for compatibility with standard Go error handling.
    *   **Role:** Enables consistent error logging, checking (e.g., `errors.Is(err, apperrors.ErrNotFound)`), and mapping to API responses (e.g., `internal/api/response.go` uses it to generate appropriate HTTP status codes and error JSON).

### 4.3. Metrics (`internal/metrics/`)

*   **Purpose:** Defines structures and functions related to application metrics, both for Prometheus scraping and direct database querying.
*   **`prom.go` (`internal/metrics/prom.go`):**
    *   **Purpose:** Sets up and exposes Prometheus metrics (Counters, Gauges, Histograms) for monitoring application behavior, particularly LLM API interactions (e.g., `LLMRequestsTotal`, `LLMFailuresTotal`, `LLMDurationSeconds`).
    *   **Usage:** Provides functions like `IncLLMRequest`, `IncLLMFailure`, `ObserveLLMDuration` to be called from relevant parts of the LLM interaction code (`internal/llm/service_http.go`). Requires `InitLLMMetrics` call at application startup (typically in `cmd/server/main.go`) and a Prometheus scrape endpoint (usually `/metrics`, though the specific handler for Prometheus metrics might be part of a library or custom setup in `main.go`).
*   **`metrics.go` (`internal/metrics/metrics.go`):**
    *   **Purpose:** Defines structs for aggregated database-derived metrics (e.g., `ValidationMetric`, `FeedbackSummary`, `UncertaintyRate`) and functions to query corresponding DB tables or views (e.g., `GetValidationMetrics`, `GetFeedbackSummary`). These are distinct from the real-time Prometheus metrics.
    *   **Usage:** These functions are typically called by specific API endpoints under `/metrics/*` (e.g., `/metrics/validation`, `/metrics/feedback`) defined in `cmd/server/main.go` to provide summary statistics based on persisted data.

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
  - Contains a `UNIQUE(article_id, model)` constraint. This is critical for enabling `ON CONFLICT` clauses in SQL queries, allowing scores to be correctly upserted (updated if they exist, inserted if new) during ensemble score updates.
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
   - Verify API key in your `.env` file (created from `.env.example`)
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