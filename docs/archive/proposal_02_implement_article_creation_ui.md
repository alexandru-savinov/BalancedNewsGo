# PR Proposal 2: Implement Article Creation UI

**Description:** This Pull Request proposes adding a user interface for manually creating new articles in the NewsBalancer system.

*   **Problem:**
    *   The backend API provides an endpoint `POST /api/articles` for creating new articles.
    *   Currently, there is no UI functionality to utilize this endpoint, meaning articles can only be added via RSS feeds or direct API calls.

*   **Proposed Solution:**
    *   **Frontend (New HTML/JS or modification to existing admin section if available):**
        *   Create a new page or a section in an existing admin panel (e.g., `/admin/create-article`).
        *   Design a form with input fields for:
            *   Article URL (string, required, validated for http/https prefix)
            *   Title (string, required)
            *   Content (text area, required)
            *   Source (string, required)
            *   Publication Date (date picker, required, formatted as RFC3339 for the API).
        *   Implement JavaScript to:
            *   Handle form submission.
            *   Validate input fields on the client-side for basic correctness.
            *   Construct a JSON payload matching the `CreateArticleRequest` schema (`source`, `pub_date`, `url`, `title`, `content`).
            *   Make a `POST` request to `/api/articles` with the payload.
            *   Handle the API response: display success messages (e.g., with the new article ID) or error messages (e.g., for duplicate URL, validation errors, server errors).
    *   **Backend (`internal/api/api.go`, `internal/api/docs/swagger.json`):**
        *   No changes are strictly required if the `POST /api/articles` endpoint (`createArticleHandler`) and its Swagger documentation are already complete and functional as per `internal/api/api.go` and `internal/api/docs/swagger.json`.
        *   A review of `createArticleHandler` for robustness and user feedback (e.g., clear error messages for conflicts) would be beneficial.

*   **Files to be Created/Modified:**
    *   New HTML file for the creation form (e.g., `web/admin_create_article.html`).
    *   New JavaScript file for the form logic (e.g., `web/js/admin_create_article.js`).
    *   Modifications to server-side routing in `cmd/server/main.go` if a new HTML page is served directly.
    *   CSS updates for styling the new form.

*   **Rationale/Benefit:**
    *   Allows authorized users (e.g., administrators, editors) to manually add articles that might be missed by RSS feeds or to quickly input specific articles for analysis.
    *   Provides a direct way to populate the system with content for testing or demonstration purposes.
