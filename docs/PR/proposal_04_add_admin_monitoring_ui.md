# PR Proposal 4: Add System Administration & Monitoring UI

**Description:** This Pull Request proposes the creation of a basic administration and monitoring section in the UI to provide insights into system health and offer manual control over certain processes.

## 1. RSS Feed Health Status Display

*   **Problem:**
    *   The backend API offers an endpoint `GET /api/feeds/healthz` that returns the health status of configured RSS feeds.
    *   This information is valuable for monitoring the data ingestion pipeline but is not currently visible in the UI.

*   **Proposed Solution (Frontend - New Admin Page/Section):**
    *   **UI:** Create a new page (e.g., `/admin/status`) or a dedicated section within an existing admin panel.
    *   **Functionality:**
        *   On page/section load, make a `GET` request to `/api/feeds/healthz`.
        *   The response is a map of feed names to boolean health statuses.
        *   Display this information in a clear list or table format, indicating which feeds are healthy (e.g., green check) and which are unhealthy/failing (e.g., red cross).
        *   Consider adding a timestamp for when the health check was last performed by the backend if available, or when the frontend fetched it.

*   **Files to be Created/Modified:**
    *   New HTML file for the admin/status page (e.g., `web/admin_status.html`).
    *   New JavaScript file for its logic (e.g., `web/js/admin_status.js`).
    *   CSS updates for styling.
    *   Server-side routing in `cmd/server/main.go` if a new HTML page is directly served.

*   **Rationale/Benefit:**
    *   Provides administrators with a quick overview of the RSS feed ingestion status.
    *   Helps in early detection of problems with feed sources or the fetching mechanism.

## 2. Manual RSS Feed Refresh Trigger

*   **Problem:**
    *   The backend API has an endpoint `POST /api/refresh` to manually trigger a refresh of all RSS feeds.
    *   There is no UI control to initiate this process manually; refreshes rely on scheduled tasks or direct API calls.

*   **Proposed Solution (Frontend - Admin Page/Section):**
    *   **UI:** On the same admin/status page proposed above (or a similar admin section), add a button labeled "Refresh All Feeds" or "Trigger Manual RSS Fetch".
    *   **Functionality:**
        *   When the button is clicked, make a `POST` request to `/api/refresh`.
        *   Provide user feedback: disable the button during the request, show a loading indicator, and display a success message (e.g., "Feed refresh started") or an error message upon completion.
        *   Since this is an asynchronous operation on the backend, the UI should indicate that the process has been *triggered*, not that it has *completed*.

*   **Files to be Created/Modified:**
    *   `web/admin_status.html` (or equivalent HTML file for the admin section) - add the button.
    *   `web/js/admin_status.js` (or equivalent JS file) - add event listener and fetch call for the button.
    *   CSS updates for styling.

*   **Rationale/Benefit:**
    *   Allows administrators to manually update the article database with the latest news on demand, without waiting for the next scheduled fetch.
    *   Useful for testing the feed fetching mechanism or quickly ingesting new content after configuration changes. 