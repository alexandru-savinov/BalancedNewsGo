# Refactoring Plan: Eliminate Environment Variable Reliance for Scoring Models

**1. Goal:**
Refactor the codebase to remove all reliance on the `LLM_DEFAULT_MODEL` and `LLM_ESCALATION_MODEL` environment variables for selecting language models during scoring operations. All scoring processes, including initial scoring and reprocessing, must exclusively use the models defined in `configs/composite_score_config.json`.

**2. Analysis Summary:**
*   **Memory Bank Context:** The system utilizes an ensemble scoring approach (`internal/llm/ensemble.go`), relies on `configs/composite_score_config.json` for model definitions, and has specific logic for reprocessing articles that fail scoring (`cmd/server/main.go`).
*   **Code Analysis:**
    *   `cmd/server/main.go`: Uses `os.Getenv("LLM_DEFAULT_MODEL")` and `os.Getenv("LLM_ESCALATION_MODEL")` within its reprocessing logic (around lines 135-155) to select a model based on the article's `FailCount`. It includes hardcoded fallbacks (`gpt-3.5-turbo`, `gpt-4`) if the environment variables are unset.
    *   `internal/llm/llm.go`: Uses `os.Getenv("LLM_DEFAULT_MODEL")` in `NewService` (around line 414) to set a default model for the LLM client, with provider-specific fallbacks. It also checks for this default model in the `Analyze` function (around line 500), returning an error if it's not set.
    *   `internal/llm/http_llm_live_test.go`: Mentions `LLM_DEFAULT_MODEL` in comments related to test setup.

**3. Refactoring Strategy:**

*   **`internal/llm/llm.go`:**
    *   **Remove `defaultModel`:** Delete the `defaultModel` field from the `Service` struct. The concept of a single default model for the base LLM service is incompatible with the ensemble approach defined in the config.
    *   **Remove Env Var Logic in `NewService`:** Delete the code block (around lines 414-428) that reads `LLM_DEFAULT_MODEL` and sets provider-specific defaults. The service initialization should focus on setting up the client connection, not selecting a default model.
    *   **Remove or Refactor `Analyze` Function:** The `Analyze` function seems tied to the old single-model approach.
        *   **Option A (Preferred):** Remove the `Analyze` function entirely if it's no longer used or if its functionality is fully superseded by the ensemble scoring methods (e.g., `ScoreArticleWithEnsemble` or similar in `internal/llm/ensemble.go`). Update any callers to use the ensemble methods instead.
        *   **Option B:** If `Analyze` is still needed for non-scoring tasks, refactor it to accept the specific model name as a parameter instead of relying on `s.defaultModel`. Remove the check `if s.defaultModel == ""` (around line 500).
*   **`cmd/server/main.go`:**
    *   **Remove Env Var Logic:** Delete the code block (around lines 138-154) that reads `LLM_DEFAULT_MODEL` and `LLM_ESCALATION_MODEL` and selects a `model` variable based on `FailCount`.
    *   **Adapt Reprocessing Logic:** Modify the reprocessing logic (likely within the loop iterating through articles needing reprocessing). Instead of calling a generic `llmService.Analyze` with a model selected via environment variables, it should invoke the **ensemble scoring mechanism**. This likely involves calling a function like `ensembleService.ScoreArticle(ctx, article)` (assuming an `ensembleService` instance is available). This ensures reprocessing uses the same configured set of models as initial scoring. Any logic related to `FailCount` (e.g., using different prompts or internal ensemble strategies for retries) should be handled *within* the ensemble service or its configuration, not by selecting different single models in `main.go`.
*   **`internal/llm/ensemble.go` (Verification):**
    *   **Verify Config Reliance:** Confirm that all functions responsible for scoring articles (e.g., `ScoreArticleWithEnsemble`, `ComputeCompositeScore`) load and use *only* the models specified in the `CompositeScoreConfig` struct, which should be populated from `configs/composite_score_config.json`.
    *   **No Implicit Defaults:** Ensure there's no hidden fallback to a "default model" concept within the ensemble logic itself.
*   **Configuration Loading:**
    *   **Verify Early Load:** Confirm that `configs/composite_score_config.json` is loaded successfully during application startup (likely in `cmd/server/main.go` when initializing services) and that the resulting configuration object is correctly passed to and stored within the ensemble service instance.

**4. Handling Edge Cases (`configs/composite_score_config.json`):**

*   **Missing or Invalid File:** If `configs/composite_score_config.json` cannot be found, read, or parsed correctly during application startup, the application **must log a fatal error and exit**. Scoring is a core function, and proceeding without a valid configuration is unsafe.
*   **Empty `models` Array:** If the configuration file is loaded successfully but the `models` array within it is empty:
    *   The ensemble scoring function(s) should detect this condition.
    *   Log a clear warning message indicating that no scoring models are configured.
    *   Return a specific error (e.g., `ErrNoModelsConfigured`) or a designated result indicating scoring was skipped.
    *   The calling code (e.g., the reprocessing loop in `cmd/server/main.go`) must handle this error gracefully (e.g., log it, skip score updates for the article).

**5. Conceptual Flow Change (Mermaid Diagram):**

```mermaid
graph TD
    subgraph Before Refactoring
        A[cmd/server/main.go Reprocessing] -- Reads --> B(LLM_DEFAULT_MODEL / LLM_ESCALATION_MODEL Env Vars);
        B -- Selects --> C{Single Model Name};
        A -- Calls Analyze with model --> D[internal/llm/llm.go Service.Analyze];
        D -- Uses --> C;
    end

    subgraph After Refactoring
        E[cmd/server/main.go Reprocessing] -- Calls ScoreArticle --> F[internal/llm/ensemble.go EnsembleService.ScoreArticle];
        G[configs/composite_score_config.json] -- Loaded into --> H(CompositeScoreConfig Struct);
        F -- Uses --> H;
        F -- Calls specific models via --> I[internal/llm/llm.go (Base Client)];
    end

    style G fill:#f9f,stroke:#333,stroke-width:2px
    style H fill:#ccf,stroke:#333,stroke-width:1px
```

**6. Documentation:**
This plan outlines the necessary steps to refactor the scoring mechanism.