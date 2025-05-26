# PR Proposal 5: Implement Content Moderation & Re-evaluation UI

**Description:** This Pull Request proposes adding UI features for content moderation, allowing authorized users to trigger re-analysis of articles, monitor scoring progress, and manually override scores.

## 1. Trigger Article Re-analysis

*   **Problem:**
    *   The backend API (`POST /api/llm/reanalyze/{id}`) allows triggering a new LLM analysis for a specific article.
    *   This is useful if an initial analysis is suspect or if LLM models/configurations have been updated, but no UI mechanism exists to invoke it.

*   **Proposed Solution (Frontend - `web/js/article.js`, `article.html` - Admin/Moderator View):**
    *   **UI:** On the article detail page (`article.html`), for users with appropriate permissions (admin/moderator), add a button like "Re-evaluate Bias" or "Request New Analysis".
    *   **Functionality (`web/js/article.js`):
        *   When the button is clicked, make a `POST` request to `/api/llm/reanalyze/{id}` (where `id` is the current article's ID). The body of the request might be empty or could optionally include parameters to influence re-analysis if the backend supports it (e.g., specific models to use, though current `reanalyzeHandler` primarily re-triggers based on existing config).
        *   Upon a successful API response (e.g., 202 Accepted indicating reanalysis started), initiate progress monitoring (see next feature).
        *   Provide feedback (e.g., "Reanalysis request sent. Monitoring progress...").

*   **Files to be Modified:**
    *   `web/js/article.js` (add event listener, fetch call for re-analysis)
    *   `web/html/article.html` (add the button, potentially conditionally rendered based on user role)
    *   CSS for styling.

*   **Rationale/Benefit:**
    *   Empowers moderators/admins to ensure analysis quality and update scores as needed.
    *   Facilitates reprocessing of articles after system updates (e.g., new LLM models, updated prompts).

## 2. Real-time Scoring Progress Monitoring

*   **Problem:**
    *   Article re-analysis can be a time-consuming process.
    *   The backend provides an SSE (Server-Sent Events) endpoint `GET /api/llm/score-progress/{id}` to stream real-time progress updates for article scoring.
    *   The UI currently does not utilize this to provide feedback on ongoing analysis tasks.

*   **Proposed Solution (Frontend - `web/js/article.js`, `article.html`):**
    *   **UI:** When a re-analysis is triggered (or if an article is loaded and found to be in an "InProgress" state), display a progress indicator section on the article detail page. This could be a progress bar, textual status updates (e.g., "Step: Starting", "Message: Starting analysis with model X"), or a combination.
    *   **Functionality (`web/js/article.js`):
        *   After successfully triggering a re-analysis via `POST /api/llm/reanalyze/{id}`, or if an article is loaded with a status indicating it's being processed:
            *   Establish an `EventSource` connection to `GET /api/llm/score-progress/{id}`.
            *   Listen for `message` events from the SSE stream.
            *   Parse the JSON data from each event (which should be a `models.ProgressState` object: `Status`, `Step`, `Message`, `Percent`).
            *   Update the UI elements (progress bar, status text) dynamically based on the received data.
            *   Handle `error` events from the `EventSource`.
            *   Close the `EventSource` connection when the `ProgressState.Status` indicates completion ("Complete", "Success", "Error", "Skipped") or if the user navigates away.
            *   If analysis completes successfully, update the article's score and confidence on the page, potentially by re-fetching the main article data or parts of it.

*   **Files to be Modified:**
    *   `web/js/article.js` (implement `EventSource` logic, UI update functions for progress)
    *   `web/html/article.html` (add HTML elements for the progress display area)
    *   CSS for styling the progress indicators.

*   **Rationale/Benefit:**
    *   Greatly improves user experience by providing real-time feedback on long-running analysis tasks.
    *   Keeps the user informed about the status and outcome of the re-analysis.

## 3. Manual Score Override

*   **Problem:**
    *   There might be cases where automated LLM analysis is incorrect or requires manual adjustment by a trusted user.
    *   The backend API `POST /api/manual-score/{id}` allows setting an article's bias score directly.
    *   No UI exists for this moderation capability.

*   **Proposed Solution (Frontend - `web/js/article.js`, `article.html` - Admin/Moderator View):**
    *   **UI:** On the article detail page, for users with appropriate permissions, add a section/modal for "Manual Score Adjustment".
        *   This section should include an input field or a slider for the score (validated between -1.0 and 1.0).
        *   A submit button "Set Manual Score".
        *   Note: The backend `manualScoreHandler` sets confidence to 1.0 for manual scores.
    *   **Functionality (`web/js/article.js`):
        *   When the "Set Manual Score" button is clicked:
            *   Get the score value from the input/slider.
            *   Perform client-side validation for the score range.
            *   Make a `POST` request to `/api/manual-score/{id}` with a JSON body like `{"score": 0.5}`.
            *   Handle the API response: display success/error messages.
            *   Upon success, update the displayed article score and confidence on the page (score to the new manual score, confidence to 100%).

*   **Files to be Modified:**
    *   `web/js/article.js` (add event listener, validation, fetch call for manual score submission, UI update logic)
    *   `web/html/article.html` (add HTML elements for the manual score input and button, conditionally rendered)
    *   CSS for styling.

*   **Rationale/Benefit:**
    *   Provides an essential moderation tool for correcting or curating article scores.
    *   Allows for human oversight and intervention in the automated scoring process.
