# PR Proposal 3: Enhance Article Detail Page with Specific Bias & Summary Sections

**Description:** This Pull Request proposes enhancing the article detail page (`article.html`, `web/js/article.js`) by integrating dedicated sections for detailed bias analysis and article summaries, utilizing specific backend endpoints.

## 1. Detailed Bias Analysis Section

*   **Problem:**
    *   The current article detail page displays an overall composite score and ensemble details.
    *   The backend provides a specific endpoint `GET /api/articles/{id}/bias` which can return individual model scores (not just the ensemble) and supports server-side filtering by score range and sorting. This offers a more granular view than currently presented directly.

*   **Proposed Solution (Frontend - `web/js/article.js`, `article.html`):**
    *   **UI:** Add a new expandable section on the article detail page titled "Detailed Bias Breakdown" or similar.
    *   **Functionality (`web/js/article.js`):
        *   When the section is expanded (or on page load if preferred), make a `GET` request to `/api/articles/{id}/bias`.
        *   Display the `results` array from the response. Each item typically contains `model`, `score`, `confidence`, `explanation`, and `created_at`.
        *   Present this data in a clear, tabular, or card-based format.
        *   **Optional Enhancements:** Add UI controls (dropdowns, sliders) to allow the user to dynamically change the `min_score`, `max_score`, and `sort` query parameters for the `/api/articles/{id}/bias` call and re-fetch/re-render the data.
    *   **HTML (`article.html`):** Add placeholder elements for this new section and its controls.

*   **Files to be Modified:**
    *   `web/js/article.js` (new fetch call, data rendering logic, event handlers for controls)
    *   `web/html/article.html` (new HTML structure for the section)
    *   CSS updates for styling.

*   **Rationale/Benefit:**
    *   Provides users with deeper transparency into the bias analysis by showing individual model scores.
    *   Leverages the filtering and sorting capabilities of the existing `/api/articles/{id}/bias` endpoint for a more interactive experience.

## 2. Dedicated Article Summary Section

*   **Problem:**
    *   The `web/js/article.js` currently seems to extract summary information from the general `/api/articles/{id}` endpoint or potentially within the ensemble details. The documentation (`codebase_documentation.md`) notes that the `SummaryHandler` in `internal/api/api.go` (serving `/api/articles/{id}/summary`) fetches scores and looks for `ModelSummarizer` metadata.
    *   A dedicated backend endpoint `GET /api/articles/{id}/summary` exists to provide a specific summary for an article.

*   **Proposed Solution (Frontend - `web/js/article.js`, `article.html`):**
    *   **UI:** Add a clearly demarcated section on the article detail page titled "Article Summary".
    *   **Functionality (`web/js/article.js`):
        *   In the `loadArticle` function (or a new dedicated function called from it), make a `GET` request to `/api/articles/{id}/summary`.
        *   The response is expected to contain `summary` text and `created_at`.
        *   Display the fetched `summary` text in the new UI section. If no summary is available (e.g., 404 from API), display an appropriate message like "Summary not available for this article."
    *   **HTML (`article.html`):** Add a placeholder element for the summary text.

*   **Files to be Modified:**
    *   `web/js/article.js` (new fetch call, logic to display summary or 'not available' message)
    *   `web/html/article.html` (new HTML structure for the summary section)
    *   CSS updates for styling.

*   **Rationale/Benefit:**
    *   Ensures that the specific, potentially curated or distinct, summary from the `ModelSummarizer` is displayed.
    *   Simplifies frontend logic by using a dedicated endpoint for summaries rather than parsing it from broader data structures.
    *   Improves content organization on the article detail page. 