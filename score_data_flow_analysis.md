# CompositeScore Data Flow Analysis

This document traces the end-to-end data flow of the `CompositeScore` from its display in the UI back to its initial ingestion from RSS feeds. The goal is to provide a clear reference for debugging issues and identifying opportunities for improvement at each stage.

---

## 1. Score Display

**Description:**  
Rendered in `<span class="composite-score">`. Formatting (`toFixed(2)`) applied via JavaScript (`loadBiasData`) for the detail view.

### Potential Debugging Points:
- Check for JavaScript errors in the browser console.
- Ensure CSS is not hiding the `.composite-score` span.
- Verify consistency between list and detail rendering logic.
- Confirm correct data binding to the DOM element.

### Opportunities for Improvement:
- Add a visual indicator of score confidence or reliability.
- Use a clearer label or tooltip to explain the score.
- Unify and refactor rendering logic for consistency across views.

---

## 2. Frontend Data Retrieval

**Description:**  
Main article list uses htmx (`hx-get="/articles"`) on page load or filter. Detail view uses JavaScript fetch or htmx to `/articles/{id}`. Backend sends JSON with `CompositeScore` (float64).

### Potential Debugging Points:
- Inspect browser network tab for API calls and responses.
- Check for htmx errors or failed requests.
- Look for JavaScript fetch errors in the console.
- Ensure the correct API endpoint is being called.

### Opportunities for Improvement:
- Implement frontend caching for article data.
- Add loading indicators for better UX.
- Display user-friendly error messages if data fetch fails.

---

## 3. Backend API Endpoint(s)

**Description:**  
`getArticlesHandler` (`/articles`) and `getArticleByIDHandler` (`/articles/{id}`) in `internal/api/api.go`. Both return JSON including `CompositeScore`. The detail endpoint also returns raw perspective scores.

### Potential Debugging Points:
- Check API logs for errors (especially 5xx responses).
- Verify the structure of the JSON response.
- Ensure the correct article ID is used in requests.
- Monitor database query performance within the handler.

### Opportunities for Improvement:
- Add API response caching for frequently accessed data.
- Strengthen input validation for API parameters.
- Standardize error responses for easier frontend handling.
- Tune pagination and filtering parameters for performance.

---

## 4. Final Score Calculation/Transformation (Backend)

**Description:**  
`llm.ComputeCompositeScore` is called in API handlers. It takes raw perspective scores (Left/Center/Right), averages them (missing scores default to 0), and calculates `1.0 - abs(average)`. This favors scores near 0 (balanced).

### Potential Debugging Points:
- Verify the input scores passed to the function.
- Check if defaulting missing scores to 0 is appropriate.
- Step through the calculation logic for correctness.
- Watch for NaN or Inf results in edge cases.

### Opportunities for Improvement:
- Explore alternative ranking or aggregation algorithms.
- Make score calculation logic configurable.
- Add a score confidence metric to the output.
- Adjust handling of missing perspective scores for accuracy.

---

## 5. Data Storage

**Description:**  
Raw perspective scores are stored in the `llm_scores` table (SQLite). Columns: `article_id` (FK), `model` (TEXT: "left"/"center"/"right"), `score` (REAL: -1.0 to +1.0).

### Potential Debugging Points:
- Check database connection and health.
- Verify the `llm_scores` table schema is correct.
- Look for missing scores for a given `article_id`.
- Assess the performance of the `article_id` index.

### Opportunities for Improvement:
- Optimize indexing strategy on `llm_scores`.
- Consider implementing score versioning/history.
- Develop a data pruning strategy for old or unused scores.
- Evaluate using a different database for better scaling.

#### Pruning and Scaling Strategies

- **Pruning:** Old or superseded score records can be pruned by deleting or archiving entries with lower `version` numbers for each `article_id`, or by removing records older than a set retention period. This can be automated with scheduled jobs or manual SQL scripts.
- **Scaling:** For larger datasets, ensure the composite index on (`article_id`, `version`) is present to optimize queries. If data volume exceeds SQLite's practical limits, consider migrating to a scalable database (e.g., PostgreSQL) and use partitioning or sharding strategies for the `llm_scores` table.

---

## 6. Data Processing/Aggregation

**Description:**  
Background job `cmd/score_articles/main.go` runs `LLMClient.ProcessUnscoredArticles()`. This calls `LLMClient.AnalyzeAndStore()`, which uses `OpenAILLMService.AnalyzeWithPrompt` for each perspective (L/C/R) and saves results to `llm_scores` via `db.InsertLLMScore`.

### Potential Debugging Points:
- Check job logs (`cmd/score_articles`) for errors (LLM API, DB writes, panics).
- Verify job scheduling and execution.
- Monitor LLM API key validity and quota.
- Examine the `metadata` column in `llm_scores` for LLM response details.

### Opportunities for Improvement:
- Add robust error handling and retries for LLM calls.
- Batch API calls and enable parallel processing of articles.
- Make LLM models and prompts configurable.
- Monitor API usage costs and optimize accordingly.
- Improve parsing of LLM responses (see `progress.md` plan).

---

## 7. Raw Data Origin

**Description:**  
Article content is fetched from a list of RSS feeds defined in `cmd/fetch_articles/main.go`.

### Potential Debugging Points:
- Verify that RSS feed URLs are valid and active.
- Check for network issues when reaching feeds.
- Examine parsing logic for compatibility with specific feed formats.

### Opportunities for Improvement:
- Add more diverse and reputable feed sources.
- Allow dynamic configuration of feed sources.
- Implement health checks for feed availability.
- Improve content extraction and normalization from feeds.

---

## 8. Initial Data Ingestion & Calculation

**Description:**  
`cmd/fetch_articles/main.go` triggers `(*Collector).FetchAndStore()` in `internal/rss/rss.go`. This stores article data (title, URL, content, etc.) in the `articles` table via `db.InsertArticle`.

### Potential Debugging Points:
- Check job logs (`cmd/fetch_articles`) for errors.
- Verify job scheduling and execution.
- Ensure database write permissions and check for write errors.
- Examine logic for handling duplicate articles.

### Opportunities for Improvement:
- Enhance duplicate detection and prevention.
- Improve error handling during fetch and store operations.
- Add more detailed logging for traceability.
- Optimize performance for high feed volume scenarios.

---

## LLM Model "Unsupported Model" Error Analysis (April 2025)

### Problem Statement
Scoring attempts for article 788 (and others) fail due to "unsupported model" errors for the left, center, and right models. The root cause is traced to the LLM backend not supporting the requested models.

### Initial Debugging Steps
- Reflected on 5-7 possible sources of the problem:
  1. LLM service configuration does not list or enable the required models.
  2. Mapping from "left", "center", "right" to actual model names is missing, incorrect, or not loaded.
  3. Code in `internal/llm/` does not register or expose the required models to the scoring pipeline.
  4. Scoring pipeline requests model names that do not match those configured/available in the LLM backend.
  5. Typo or mismatch in model names between config and code.
  6. LLM client is initialized with a restricted set of models, omitting the needed ones.
  7. Configuration file (e.g., in `configs/`) is missing entries for the required models.

- Most likely sources:
  - (1) The LLM service configuration does not list or enable the required models.
  - (2) The mapping from "left", "center", "right" to actual model names is missing or incorrect.

### Codebase Observations
- `internal/llm/llm.go` uses constants for `LabelLeft`, `LabelRight`, and `LabelNeutral`, and in `ComputeCompositeScore`, it matches on `s.Model` being "left", "center", or "right".
- The actual mapping or registration of which model names are available, and how "left", "center", "right" are mapped to real model names, is not yet visible. This is likely handled elsewhere in `llm.go` or in a config file (possibly in `configs/`).

### Next Steps
- Inspect more of `llm.go` for model registration and selection logic.
- Check for configuration files in `configs/` that define available models or mappings.
- Add logs to the model selection logic to print out which models are available and which are being requested, to validate the diagnosis.

---

## Resolution & Production Deployment (April 2025)

### Issue Resolution
- The scoring pipeline issues were resolved by:
  - Correcting the mapping between logical model labels ("left", "center", "right") and the actual LLM model names in both configuration and code.
  - Ensuring all required models were registered and available in the LLM backend.
  - Verifying that the scoring job (`cmd/score_articles/main.go`) executed successfully for all articles, including those previously affected by model mapping errors.

### Verification
- Article 788, which previously failed to receive a score due to model mapping issues, now displays a real, nonzero `CompositeScore` on the main page.
- Manual and automated checks confirmed that scores are now computed and displayed correctly for all main page articles.

### Production-Ready Status
- As of April 11, 2025, the scoring system is fully functional and production-ready.
- The end-to-end pipeline (article selection, LLM model mapping, scoring job execution, frontend display) has been verified in a production environment.
- Ongoing monitoring and documentation updates will continue as the system evolves.