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

## Database Layer (`internal/db/`)

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

## Web Interface (`web/` & Editorial Template Integration)

*   **Editorial Template Enhancements:**
    *   Optimize template caching strategies to further improve the already excellent 2-20ms response times
    *   Add template customization options for different themes or layouts
    *   Implement template-level A/B testing capabilities
*   **API Integration:**
    *   Implement the missing `/api/sources` endpoint to support dynamic source filtering in the UI (currently the endpoint returns 404 as seen in the server logs).
    *   Add proper error handling for all API calls with user-friendly error messages and retry mechanisms for transient failures.
*   **User Experience:**
    *   Add a source management page for administrators to add, edit, or remove news sources.
    *   Implement user accounts and saved preferences (favorite sources, preferred sorting options, etc.).
    *   Add a search functionality to find articles by keywords, topics, or content.
    *   Enhance the feedback system to allow users to see aggregated feedback from other users.
*   **Performance:**
    *   Further optimize client-side JavaScript enhancement to complement the fast server-side rendering
    *   Implement service workers for offline capabilities and faster loading times.
    *   Add lazy loading for article contents in the list view to improve initial page load times.
*   **Visualization:**
    *   Create more advanced visualizations for bias analysis, such as historical trends for sources or topics.
    *   Add interactive charts showing the distribution of political leanings across different news sources.
    *   Implement a "balance view" feature that presents articles from different perspectives on the same topic.
*   **Accessibility:**
    *   Conduct a thorough accessibility audit and implement WCAG 2.1 AA compliance.
    *   Add keyboard navigation support for all interactive elements.
    *   Ensure proper contrast ratios and text scaling for better readability.
*   **Responsiveness:**
    *   Further optimize the already excellent mobile-responsive Editorial template design
    *   Implement a progressive web app (PWA) configuration for better mobile integration.
*   **Testing:**
    *   Develop comprehensive end-to-end tests for the Editorial template interface using Playwright or similar tools.
    *   Add automated visual regression testing to catch unexpected UI changes in template rendering.
    *   Create user testing scenarios to validate template usability improvements.
*   **Code Structure:**
    *   Consider migrating client-side JavaScript enhancements to TypeScript for improved type safety and developer experience.
    *   Organize template assets using a modern build system for optimized delivery.