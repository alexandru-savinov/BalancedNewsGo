# Potential Codebase Improvements

This document outlines potential areas for improvement and future development within the NewsBalancer Go codebase, categorized by component. These points were identified during a codebase review and documentation process.

## API Layer (`internal/api/`)

*   **Caching:** While `/api/articles` is cached, consider extending caching to other frequently read, idempotent endpoints like `/api/articles/:id`, `/api/articles/:id/bias`, `/api/articles/:id/summary`, `/api/articles/:id/ensemble`, and `/api/feeds/healthz` using the existing `SimpleCache` mechanism or a similar approach.
*   **Input Validation:**
    *   Payload validation using `binding:"required"` is good. Enhance this with more specific struct tags for ranges (e.g., score values) and formats where applicable (e.g. `binding:"min=-1,max=1"` if supported, or manual validation post-bind).
    *   Implement explicit validation for query parameters (e.g., ensuring `limit` > 0, `offset` >= 0) in handlers like `getArticlesHandler`.
*   **Error Responses:** The standardized error handling via `SafeHandler`, `RespondError`, and `apperrors.AppError` is well-implemented. Continue ensuring its consistent application across all handlers and error paths.
*   **Pagination and Filtering:**
    *   The existing pagination (`offset`, `limit`) and source filtering for `/api/articles` is a good foundation.
    *   Introduce API parameters for sorting (e.g., `sort_by=pub_date&order=desc`) on endpoints like `/api/articles`.
    *   Evaluate adding filters for other fields (e.g., date ranges, score ranges) based on user needs.
    *   Ensure database queries are optimized for these parameters (see DB improvements).
*   **Code Structure:** Review and refactor or remove files with minimal or commented-out code within the `internal/api/` package (e.g., `handlers.go`, `db_operations.go` if they remain unused) to streamline the codebase.
*   **Progress Tracking Unification:** Examine the progress tracking mechanisms used by the API layer (if any custom map exists, as hinted in documentation) and `internal/llm/progress_manager.go`. Unify or clarify their interaction to ensure a single source of truth and avoid redundancy.

## LLM Analysis Core (`internal/llm/`)

*   **Error Handling & Retries:**
    *   The existing error handling for rate limits (including backup keys and model fallbacks in `service_http.go`) is robust. The retry loop in `ensemble.go` also improves resilience.
    *   Consider making retry attempts and backoff strategies (currently `maxRetries := 2` in `ensemble.go`) configurable, potentially via `CompositeScoreConfig`.
*   **Batching:** Investigate if the LLM provider (e.g., OpenRouter) supports batch API calls. If so, implementing batch processing for multiple articles/contents could improve throughput and potentially reduce costs. (Currently, calls are per-article).
*   **Configuration:**
    *   LLM models are well-managed via `CompositeScoreConfig`.
    *   Enhance prompt management by allowing prompt variants (templates, examples, currently from `internal/llm/configs/*.txt`) to be defined or referenced within `CompositeScoreConfig.json` for more centralized configuration.
*   **Cost & Usage Monitoring:**
    *   The Prometheus metrics (`LLMRequestsTotal`, etc.) provide a good starting point.
    *   For deeper cost optimization, explore logging token usage (if available from LLM API responses) and associating costs with specific models/prompts. This can inform more advanced model selection strategies beyond rate-limit fallbacks.
*   **Response Parsing:** The `parseNestedLLMJSONResponse` in `ensemble.go` (handling JSON, Markdown-JSON, and regex fallbacks) is quite robust. Continue monitoring its effectiveness and adapt if new problematic response formats emerge.
*   **Score Aggregation:**
    *   The system supports configurable aggregation (`formula`, `weights` in `CompositeScoreConfig`).
    *   To further enhance, new aggregation algorithms beyond "average" and "weighted" could be implemented and made selectable via the configuration.
*   **Score Calculation Logic:** The core calculation logic (`formula`, `weights`, `confidence_method`) is already well-parameterized through `CompositeScoreConfig`.
*   **Handling Missing/Invalid Scores:**
    *   Configuration for `default_missing` and `handle_invalid` exists.
    *   Implement the proposed improvement (see `docs/PR/handle_total_analysis_failure.md`) to return a specific error (e.g., `ErrAllPerspectivesInvalid`) when all LLM perspectives fail, instead of defaulting to a potentially misleading 0.0 score. This will allow `ScoreManager` to handle such cases more explicitly (e.g., by not persisting the score).
*   **Database Connection Management:** Ensure `LLMClient` and other components utilize the main, shared database connection pool (initialized by `db.InitDB`) rather than establishing separate connections, for efficiency and consistency.

## Database Layer (`internal/db/`)

*   **Replace `sqlx` with `sqx`:** Migrate from `github.com/jmoiron/sqlx` to `github.com/stytchauth/sqx`.
    *   **Rationale:** The primary goal is to eliminate the CGO build dependency introduced transitively by `sqlx` (via `github.com/mattn/go-sqlite3`). `sqx` and its core dependencies (`squirrel`, `blockloop/scan`) appear to be pure Go. While the standard `database/sql` package used with `modernc.org/sqlite` would also be CGO-free, `sqx` offers additional benefits like fluent query building, simplified struct scanning, and compile-time type safety for results via generics, potentially improving robustness and developer experience compared to using `database/sql` directly.
*   **Indexing Strategy:** This is an ongoing optimization task. Review and potentially add or modify indexes (e.g., compound indexes on `articles(source, pub_date, id)`) to support common query patterns observed in `FetchArticles` (filtering by source, leaning, sorting by date).
*   **Score Versioning/History:** The `LLMScore` table includes `Version` and `CreatedAt` fields. New analyses appear to insert new rows, implicitly providing a history. Ensure the `Version` field is consistently populated and consider adding utility functions if more explicit querying of score history is needed.
*   **Data Pruning:** Develop and automate a strategy (e.g., a new `cmd/` tool or a scheduled job) for pruning old or irrelevant scores and articles to manage database size and performance over time.
*   **Database Scalability:** For future significant scaling, continue to keep in mind the potential evaluation of migrating from SQLite to a more concurrent database system (e.g., PostgreSQL).

## Data Ingestion (`internal/rss/`)

*   **Feed Sources:** Regularly review and update `configs/feed_sources.json` with diverse and reputable sources.
*   **Dynamic Feed Management:** Implement API endpoints and potentially a UI for administrators to dynamically add, update, or remove RSS feed sources without requiring configuration file changes and application restarts.
*   **Health Checks:** The current `CheckFeedHealth` (verifying URL parsing) is a good baseline. Enhance this with configurable retries for transient network issues or checks for consistently empty/unchanged feeds.
*   **Content Extraction & Normalization:** The current `extractContent` method is basic. Improve it by adding HTML sanitization (to get clean text) and potentially integrating a library for boilerplate removal (ads, navigation elements) to enhance the quality of content passed to LLMs.
*   **Duplicate Detection:** URL and similar-title duplicate detection are implemented. For greater accuracy, consider adding content-based duplicate detection (e.g., using MinHash or SimHash algorithms on article content).
*   **Error Handling & Reporting:** Logging of fetch/store errors is in place. For better operational insight, consider adding specific Prometheus metrics for RSS fetching (e.g., feeds successfully fetched/failed, articles added).
*   **Logging:** Current logging provides good traceability. Minor enhancements could include unique IDs for fetch cycles if needed for very detailed debugging.
*   **Performance:** The current sequential fetching of feeds and items can be a bottleneck for a large number of sources. Explore parallelizing feed fetching (e.g., using a worker pool) and potentially batching database insertions if performance becomes an issue.
*   **`LLMClient` in `Collector`:** Evaluate the `LLMClient` dependency in the `rss.Collector`. If it's not actively used for tasks like immediate post-fetch analysis, consider removing it to simplify the component. Alternatively, define and implement functionality that leverages this client (e.g., for quick categorization or pre-filtering of fetched articles).

## Configuration Management

*   **Database Configuration:** Standardize database connection configuration. Fully utilize `DATABASE_URL` from `.env` or clearly document and justify any hardcoded database paths/names, ensuring consistency.
*   **LLM Model `role` Field:** Clarify the purpose and usage of the optional `role` field within the `models` array in `configs/composite_score_config.json`. If actively used, document its impact. If not, consider for deprecation or future functionality.
*   **`CompositeScoreConfig` Caching:** Implement caching for `CompositeScoreConfig` loaded by `internal/llm/composite_score_utils.go::LoadCompositeScoreConfig` to improve performance. Provide a mechanism to invalidate or reload the cache if dynamic updates without application restart are a requirement.

## Web Interface/Frontend

*   **Feature Enhancement:** If the current web interface (`web/`) is primarily for basic display, consider expanding its functionality. This could include more interactive data exploration, administrative tasks (e.g., managing feed sources if dynamic management is added), or richer visualization of bias scores and analysis metadata.

## Metrics and Monitoring

*   **Custom Metrics Table Population:** For any custom metrics stored in database tables (as suggested by `internal/metrics/metrics.go`), clearly define, implement, and document the mechanism for their population (e.g., scheduled background jobs, triggers). Ensure this process is robust and reliable.

## Security

*   **Comprehensive Security Review:** Conduct a thorough security audit across the application. Key areas include:
    *   **API Hardening:** Advanced input validation for all API endpoints (query params, path params, request bodies) beyond basic struct binding.
    *   **Secret Management:** Ensure API keys and other secrets are handled securely, not leaked in logs, and loaded correctly from environment variables.
    *   **Web Vulnerabilities:** If the web interface becomes more interactive, assess for common vulnerabilities (e.g., XSS, CSRF).
    *   **Database Security:** Confirm that all database interactions are secure, parameterized queries are used universally (SQLx helps), and database access permissions are appropriate.
*   **Dependency Vulnerabilities:** Implement a process for regularly scanning dependencies for known vulnerabilities and updating them.

## Deployment and Operations

*   **Standardized Deployment:** Develop and document standardized deployment procedures. This could include providing Dockerfiles, example Kubernetes manifests, or scripts for deploying the Go binary as a system service.
*   **Configuration Management in Deployment:** Provide clear guidance on managing configurations (e.g., `.env` files, JSON configs) in different deployment environments (dev, staging, prod).
*   **Logging Aggregation & Monitoring:** Recommend or integrate with centralized logging solutions (e.g., ELK stack, Grafana Loki) and set up monitoring/alerting for critical application errors and performance metrics in production. 