# NewsBalancer Minimal Front‑End Proposal (Merged v1.3 – Updated to Backend Implementation)

## Overview and Page Routes  
The NewsBalancer front-end is a multi-page web interface with server-rendered templates, aligned to specific Go backend handlers. The core pages include: **Articles Listing**, **Article Detail**, and an **Admin Dashboard**. Each page corresponds to a dedicated route and Go template, served by handlers in the backend:

- **Articles List Page** – Accessible at `/articles`, handled by `TemplateIndexHandler` in Go. This route returns the **articles listing page** (template `articles.html`) populated with a list of news articles and their bias summary data.  
- **Article Detail Page** – Accessible at `/article/:id`, handled by `TemplateArticleHandler`. This returns the **article detail page** (`article.html` template) showing the full content of a single article along with its bias analysis results.  
- **Admin Dashboard** – Accessible at `/admin` (handled by `TemplateAdminHandler`), providing summary statistics and controls for maintenance tasks (e.g. feed refresh, metrics).  

The front-end uses standard navigation for these routes (full page loads). The backend templates are rendered with real data from the database, ensuring the pages are usable even without client-side JavaScript. For example, the Article Detail template is server-rendered with the article content, its current composite bias score, a bias label (“Left Leaning”, “Right Leaning”, or “Center”), and a summary of the article (if available). Recent articles and basic stats are also included in sidebars for context. *(In the original proposal, some content was expected to load via HTMX requests; however, the implemented design renders primary content on the server to simplify initial load and SEO.)*

**Routing Details:** The root path `/` simply redirects to `/articles`. Both the list and detail pages leverage Go templates. The `TemplateIndexHandler` fetches articles from the SQLite DB applying any query filters or pagination before rendering HTML. The `TemplateArticleHandler` fetches the specified article by ID and enriches it with analysis results (composite score, confidence, bias label, summary) prior to template rendering. This alignment ensures that every front-end route has a corresponding backend handler and template, and that any dynamic data displayed has an actual source in the codebase.

*(**Note:** In earlier planning, HTMX was considered for partial content loading, but the final implementation favors full server-side rendering for primary views, with progressive enhancement via the JSON API and JavaScript for interactive features.)*

## Filtering, Search, and Pagination on `/articles`  
The articles listing page supports filtering by source, political leaning, and text search, as well as pagination. These are handled through query parameters on the same `/articles` route, mapped to backend logic:

- **Source Filter:** e.g. `?source=BBC` – Returns only articles from “BBC”. The handler adds a SQL condition `AND source = ?` when this parameter is present.  
- **Bias/Leaning Filter:** e.g. `?leaning=Left` – Filters articles by bias direction. The backend maps “Left” to composite_score < -0.1, “Right” to > 0.1, and “Center” to between -0.1 and 0.1. (`bias` is accepted as an alias for `leaning` for backward compatibility.)  
- **Search Query:** e.g. `?query=election` – Performs a substring match on title or content. The handler appends a `title LIKE ? OR content LIKE ?` filter for the query term.  
- **Pagination:** e.g. `?page=2` – Paginates results 20 per page. The backend uses `page` to calculate an `OFFSET` (`(page-1)*20`) and sets a `LIMIT` of 21 items (one extra to detect if more pages remain). In the rendered template, it sets `HasMore` if an extra item was found beyond the page limit. The front-end can then show a “Next Page” link or load-more button if `HasMore` is true. Current page number is tracked as `CurrentPage` in the template data.

These filters do not require separate AJAX endpoints – the user can navigate or submit a form and the server returns a new filtered HTML page. (Originally, a more dynamic HTMX approach was proposed for filtering without full reloads. In practice, the implemented front-end opted for standard page requests or a possible JavaScript fetch approach for simplicity.)

**Data displayed:** Each article entry in the list includes its title, source, publication date, and a bias indicator. The bias indicator is derived from the article’s `CompositeScore` and `Confidence` provided by the backend. The code calculates these values by aggregating all LLM model scores for that article (weighted by confidence). The `CompositeScore` is a number between -1 (left) and +1 (right); the template may format it to two decimal places or visually via a bias slider component. A textual **Bias Label** is also shown (e.g. “Left Leaning”, “Center”, “Right Leaning”), determined on the backend using threshold ±0.3. This ensures consistency – the label in the UI directly reflects the server’s categorization of the composite score. *(Originally, the proposal assumed slightly different bias thresholds; the implementation standardized on 0.3 for a clear distinction, which is now reflected in the front-end labels.)*

## Article Detail Page – Dynamic Analysis and SSE Updates  
The article detail view (`/article/:id`) provides the full article text and a detailed bias analysis. Upon initial load, the server has already embedded the article content and the latest bias analysis results into the HTML. This includes: the composite bias score (and corresponding label/color on the bias slider), the confidence level of that score, a short summary of the article, and possibly an initial breakdown of bias by perspective. The summary is generated by a background LLM (“summarizer”) and stored in the database; the handler fetches it and injects it if available. Recent articles are listed in a sidebar for navigation/context.

**Re-Analysis Feature:** A key interactive element on the detail page is the ability to trigger a **re-analysis** of the article’s bias. This is typically a button (e.g. “Reanalyze” or “Refresh Bias Score”). When clicked, the front-end sends a POST request to the backend endpoint **`POST /api/llm/reanalyze/:id`** with the article ID. This endpoint immediately enqueues re-analysis and returns 202 Accepted semantics. The backend logs and sets the initial progress state (status “InProgress”) and then processes the LLM calls asynchronously.

**Progress via SSE:** As the re-analysis runs, the front-end opens an SSE connection to **`GET /api/llm/score-progress/:id`**. Each event contains a JSON payload with the current `step`, `percent`, `status`, and possibly `final_score`. When the status is “Complete”, the SSE stream closes; the front-end then calls **`GET /api/articles/:id/bias`** to fetch the freshest composite score and update the UI.

## Bias Analysis Data APIs  
- **`GET /api/articles/:id/bias`** – Returns JSON with `composite_score` and `results` (per-model scores).  
- **`GET /api/articles/:id/ensemble`** – Returns detailed ensemble breakdown including sub_results and aggregation stats.

Both endpoints are cached for performance and reflect the latest stored analysis data.

## User Feedback Integration  
The detail page embeds a feedback form that posts to **`POST /api/feedback`**, sending `{article_id, user_id?, feedback_text, category?}`. On success, the backend writes a feedback record and adjusts the article’s confidence ±0.1 based on `category` (“agree”/“disagree”). The UI shows a thank‑you state and, on next refresh, displays any updated confidence.

## Admin‑Only Functions  
- **Manual Scoring:** `POST /api/manual-score/:id` with `{"score": float}` to override a bias value.  
- **RSS Refresh:** `POST /api/refresh` to trigger background feed collection.  
- **System Metrics / Health:** `/metrics/*` and `/healthz` endpoints provide JSON data used by admin dashboard charts or tiles.

## Alignment Highlights  
- All routes/UI controls map to real backend handlers.  
- JSON field names and threshold values match code (`composite_score`, 0.3 bias cutoff, `confidence` 0–1).  
- SSE progress format is captured exactly; front‑end must handle stream termination.  
- Feedback immediately tweaks `confidence`, not the score itself.  
- Filtering, search, and pagination rely on query parameters handled server‑side; no HTMX required for minimal operation.  

## Conclusion  
This updated document guarantees that the front‑end’s expectations are fully synchronized with the backend’s capabilities in branch `frontend-rewrite-v3`, ensuring developers can implement UI components without encountering missing endpoints or mismatched JSON structures. Progressive enhancement via fetch/XHR or HTMX is still possible, but the base server-rendered HTML is always functional, accessible, and SEO‑friendly.
