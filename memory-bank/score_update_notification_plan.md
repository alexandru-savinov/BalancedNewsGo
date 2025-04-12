# Plan: Redesign Scoring Update &amp; Notification Mechanism

**Date:** 2025-04-12
**Author:** Roo (Architect Mode)
**Status:** Approved

## Goal

Redesign the scoring update and notification mechanism to reliably reflect the final score in the frontend after manual re-analysis (`POST /api/llm/reanalyze/:id`). This involves ensuring the background task signals its *final* state (success + score, or error) via the existing SSE stream (`/api/llm/score-progress/:id`) for direct UI updates.

## Current Issues Addressed

1.  Premature "Storing results" progress message in `reanalyzeHandler`.
2.  Final "Complete" SSE message lacks the calculated score.
3.  Error messages via SSE lack specific status indicators.

## Proposed Changes

**1. Enhance Backend Data Structure (`ProgressState`)**

*   **File:** `internal/api/api.go`
*   **Goal:** Modify the `ProgressState` struct to carry the final status and score.
*   **Changes:**
    *   Add a `Status` field (string): "InProgress", "Success", "Error".
    *   Add a `FinalScore` field (pointer to float64, `*float64`) for the score upon success.

    ```go
    // Proposed change in internal/api/api.go
    type ProgressState struct {
        Step        string   `json:"step"`         // Current detailed step (e.g., "Scoring with model X")
        Message     string   `json:"message"`      // User-friendly message for the step
        Percent     int      `json:"percent"`      // Progress percentage
        Status      string   `json:"status"`       // Overall status: "InProgress", "Success", "Error"
        Error       string   `json:"error,omitempty"` // Error message if Status is "Error"
        FinalScore  *float64 `json:"final_score,omitempty"` // Final score if Status is "Success"
        LastUpdated int64    `json:"last_updated"` // Timestamp
    }
    ```

**2. Modify Backend Logic (`reanalyzeHandler` Goroutine)**

*   **File:** `internal/api/api.go`
*   **Goal:** Adjust the background goroutine to calculate the score *before* storing, fix progress message timing, and populate the new `ProgressState` fields correctly.
*   **Changes:**
    *   **Calculate Score Early:** After successful model scoring loop, calculate the ensemble score and store it locally.
    *   **Correct "Storing" Message Timing:** Move the `setProgress(..., "Storing results", ...)` call to *after* successful `StoreEnsembleScore`. Keep `Status` as "InProgress".
    *   **Populate Final Success State:** On final completion, call `setProgress` with `Status: "Success"` and `FinalScore` pointing to the calculated score.
    *   **Populate Final Error State:** On any error, call `setProgress` with `Status: "Error"`, the specific `Error` message, and `FinalScore: nil`.

**3. Backend SSE Handler (`scoreProgressSSEHandler`)**

*   **File:** `internal/api/api.go`
*   **Goal:** Ensure the SSE handler correctly transmits the enhanced `ProgressState`.
*   **Changes:** None required. The handler already sends the full `ProgressState`.

**4. Define Frontend Behavior (JavaScript)**

*   **Goal:** Outline how the frontend JavaScript should handle the new SSE message structure.
*   **Logic:**
    *   Connect to `/api/llm/score-progress/:id` via SSE.
    *   Listen for `message` events.
    *   Parse `event.data` JSON.
    *   Update general progress UI.
    *   **Check `Status`:**
        *   `"Success"`: Extract `FinalScore`, update UI score directly, indicate completion, close SSE.
        *   `"Error"`: Extract `Error` message, display error, indicate failure, close SSE.
        *   `"InProgress"`: Continue updating progress.

**5. Sequence Diagram**

```mermaid
sequenceDiagram
    participant FE as Frontend (JS)
    participant API as Go API (/api/llm/reanalyze/:id)
    participant BG as Background Goroutine
    participant SSE as SSE Handler (/api/llm/score-progress/:id)
    participant DB as Database
    participant LLM as LLM Client

    FE->>+API: POST /api/llm/reanalyze/{id}
    API->>API: Set Progress(id, Step="Queued", Status="InProgress")
    API->>+BG: Start Goroutine(id)
    API-->>-FE: Response {"status": "reanalyze queued"}
    FE->>+SSE: GET /api/llm/score-progress/{id} (SSE Connection)
    SSE-->>FE: Send Progress(Step="Queued", Status="InProgress")

    BG->>BG: Set Progress(id, Step="Starting", Status="InProgress")
    BG->>DB: Fetch Article(id)
    DB-->>BG: Article Data
    BG->>DB: Delete Old Scores(id)
    DB-->>BG: Success/Error
    loop For Each Model
        BG->>BG: Set Progress(id, Step="Scoring Model X", Status="InProgress")
        BG->>LLM: ScoreWithModel(article, modelX)
        LLM-->>BG: Score Result / Error
        alt Error Scoring
            BG->>BG: Set Progress(id, Step="Error", Status="Error", Error="...")
            BG-->>-SSE: Notify SSE Handler (via shared progressMap)
            SSE-->>FE: Send Progress(Status="Error", Error="...")
            FE->>FE: Display Error, Close SSE
            BG->>BG: Exit Goroutine
        end
    end
    BG->>BG: Calculate Final Ensemble Score (local variable `finalScore`)
    BG->>LLM: StoreEnsembleScore(article) # Attempts to store in DB via LLM Client
    LLM-->>BG: Success/Error
    alt Error Storing Score
        BG->>BG: Set Progress(id, Step="Error Storing", Status="Error", Error="...")
        BG-->>-SSE: Notify SSE Handler
        SSE-->>FE: Send Progress(Status="Error", Error="...")
        FE->>FE: Display Error, Close SSE
        BG->>BG: Exit Goroutine
    else Store Success
        BG->>BG: Set Progress(id, Step="Storing results", Status="InProgress") # Message after success
        BG-->>-SSE: Notify SSE Handler
        SSE-->>FE: Send Progress(Step="Storing results", Status="InProgress")
        BG->>BG: Set Progress(id, Step="Complete", Status="Success", FinalScore=finalScore)
        BG-->>-SSE: Notify SSE Handler
        SSE-->>FE: Send Progress(Status="Success", FinalScore=finalScore)
        FE->>FE: Update UI with finalScore, Close SSE
        BG->>BG: Exit Goroutine
    end