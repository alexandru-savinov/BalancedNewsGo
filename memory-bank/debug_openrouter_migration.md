# Debugging Plan: OpenRouter Migration LLM Request Failures

**Goal:** Identify and resolve the root cause of failed LLM requests to OpenRouter after migrating from OpenAI.

**Primary Hypothesis:** Leftover hardcoded OpenAI references (endpoints, model names, SDK usage, config values) exist in the codebase, configuration, or environment, causing requests to fail.

**Investigation Steps:**

1.  **Search for Hardcoded OpenAI References**
    *   1a. Search codebase (`internal/`, `cmd/`) for `api.openai.com`, OpenAI model names (e.g., `gpt-`, `text-embedding-`), and OpenAI SDK imports/usage.
        *   **Findings:**
            *   Multiple hardcoded OpenAI model names (e.g., `gpt-3.5-turbo`, `gpt-4`) found in `internal/llm/llm.go`, `internal/llm/openai_live_test.go`, `internal/llm/ensemble.go`, `internal/api/api_test.go`, `cmd/test_openai/main.go`, `cmd/server/main.go`.
            *   Logic dependent on `"openai"` provider string and `OPENAI_API_KEY` environment variable found in `internal/llm/llm.go` and `internal/llm/openai_live_test.go`.
            *   Functions/types specific to OpenAI (`callOpenAIAPI`, `OpenAILLMService`, `callOpenAI`) used in `internal/llm/ensemble.go`.
            *   No hardcoded `api.openai.com` endpoints found in the code.
    *   1b. Examine configuration files (`.env`, `configs/*.json`, `configs/*.txt`) for OpenAI endpoints, keys, provider settings, or model names.
        *   **Findings:**
            *   `.env`: Contains `OPENAI_API_KEY` variable, `LLM_PROVIDER=openai` setting, and `OPENAI_MODEL=gpt-3.5-turbo` setting.
            *   `configs/composite_score_config.json`: Contains the model name `"openai/gpt-3.5-turbo"`.
            *   `configs/bias_config.json`: No OpenAI references found.
            *   `configs/prompt_template.txt`: No OpenAI references found.
    *   1c. Check environment variables (implicitly via `.env` check in 1b, potentially ask user if needed).
        *   **Findings:** Covered by 1b.

2.  **Analyze Findings & Update Plan**
    *   Findings from Step 1 (codebase and config search) confirm that hardcoded OpenAI references (model names, provider strings, API key variable names, specific functions/types) exist in multiple locations.
    *   The primary hypothesis is supported.
    *   The next step was **Step 4: Implement Fix** to refactor these hardcoded values and logic to use configurable OpenRouter settings. Step 3 (Investigate Alternative Causes) was deemed unnecessary.

3.  **Investigate Alternative Causes (if needed)**
    *   Authentication (API Key format/validity)
    *   Request Structure (Body, Headers, Parameters, Model ID format)
    *   Endpoint URL
    *   Model Compatibility (Unsupported parameters/features)
    *   Network Connectivity (Firewall, DNS)
    *   Rate Limits/Quotas
    *   SDK/Wrapper Issues

4.  **Implement Fix**
    *   Applied the necessary code/config changes based on the findings from Step 1. This involved replacing hardcoded OpenAI model names, provider strings, and refactoring OpenAI-specific functions/types to be more generic and use OpenRouter configuration read from `.env` or config files consistently.

5.  **Verify Fix**
    *   Tested the LLM request functionality to confirm the issue is resolved.
    *   **Verification Results:**
        *   `go run ./cmd/test_llm` executed successfully.
        *   Confirmed usage of `openrouter` provider from environment variables.
        *   Successful `200 OK` API calls to OpenRouter for tested models (`openai/gpt-3.5-turbo`, `openai/gpt-4`).
        *   Successful responses received and parsed.
        *   No "Unsupported model" or other critical LLM errors observed.

6.  **Final Update & Conclusion**
    *   The root cause was confirmed to be hardcoded OpenAI references (model names, provider strings, API key variable names) which were successfully refactored in Step 4 to use environment configuration. The issue is now resolved.

**Status:** Completed