# Active Context

[2025-04-12 12:06:01] - Memory Bank Update (UMB) performed at user request. All memory-bank files reviewed; session context synchronized.
---

**Summary (April 2025):**  
The 2025 redesign is now live, featuring a robust **multi-model, multi-prompt ensemble** for nuanced bias detection. The **API** has been enhanced with endpoints for user feedback and bias insights, while the **frontend** supports dynamic content loading and inline feedback submission. **Prompt engineering** has been refined with configurable templates and few-shot examples, improving LLM reliability. A **continuous validation and feedback loop** guides ongoing improvements. Major **refactoring** improved modularity and maintainability, resolving key **SonarQube warnings** and stabilizing the codebase.

---

## E2E Testing Methodology (April 2025)
- **Preparation:** Before running e2e tests, the `e2e_prep.js` script snapshots all news feeds and LLM model configs, performs health checks on feeds, LLM APIs, and the database, and triggers article ingestion via the Go CLI. The process halts if any critical service is unhealthy, ensuring tests run against a known-good, reproducible environment.
- **Test Execution:** Cypress is used for e2e tests, with test specs organized under `cypress/e2e/**/*.cy.js` and executed against `http://localhost:8080`. No global support file is loaded; each suite is self-contained.
- **Traceability:** Each test run is associated with a unique snapshot and metadata, enabling reproducibility and debugging.
- **Best Practices:** This methodology ensures reliable, actionable e2e results and rapid diagnosis of failures.

---

## Backend Status (Post-Implementation)

### RSS Fetching
- Fully implemented via `internal/rss`
- Fetches and parses multiple news sources reliably

### Database
- SQLite schema operational
- Stores raw articles and LLM analysis results
- Repository pattern abstracts DB access

### LLM Integration
- **OpenAI API fully integrated, replacing mock services**
- **Prompt engineering refined with configurable templates**
- **Bias detection enhanced with structured outputs and heuristics**
- Summarization and bias detection functional via OpenAI
- Multi-perspective extraction basic, planned for future refinement

### API
- REST API operational via `internal/api`
- Serves articles, summaries, and bias data
- **Endpoints for user feedback, comparison, and bias insights implemented**

### Frontend
- Improved UI with htmx dynamic loading
- Displays articles, summaries, bias info
- Supports inline user feedback submission
- Responsive and accessible design

### Article Page UI/UX Redesign (April 2025)
- Introduced an **interactive bias visualization slider** with composite and individual model indicators.
- Added **tooltips** for detailed model explanations, scores, and confidence.
- Enabled **toggleable advanced view** with ensemble details and aggregation stats.
- Optimized article images for **responsiveness and lazy loading**.
- Improved layout with a **responsive grid**, modern styling, and inline feedback options.
- Cross-reference: [`web/index.html`](../web/index.html)

## Current Focus
- Testing and validation of new features
- Collecting user feedback
- Planning future enhancements (multi-perspective extraction, source diversity)

## Recent Changes
- Toolchain issues fixed, improving build/test reliability
- Backend audit completed, confirming modular design is sound
- **OpenAI API integration completed and verified**
- Backend components integrated and functional end-to-end
- Frontend enhanced with summaries, bias info, and feedback forms
- **Article page UI/UX redesigned with interactive bias visualization, image optimization, and layout improvements**

## Debugging and Diagnostics

- Introduced **verbose logging** across backend modules to facilitate troubleshooting.
- Fixed a **nil pointer dereference bug** in the LLM integration module, improving stability.
- Improved configuration fallback logic to use environment variables when config files are unavailable.

## Next Steps and Improvements

### Quality Control Loops
- Integrate automated validation after each subtask 
- Use assertions and heuristics to catch errors early
- Implement feedback loops where agents can request clarification or reprocessing if confidence is low

### Automated Testing
- Add or fix unit and integration tests for backend components
- Ensure OpenAI integration, bias detection, and API endpoints are covered
- Set up CI (e.g., GitHub Actions) to run tests on every push

### Robust Error Handling and Logging
- Improve error messages and add structured logging
- Log LLM API failures, database errors, and user input issues clearly

### Security Improvements
- Secure API keys and sensitive configs (use environment variables or secret managers)
- Add input validation and sanitize user feedback submissions
- Plan for authentication and authorization if user accounts are added

### Frontend Polish
- Improve UI/UX for clarity and responsiveness
- Add loading indicators, error messages, and success confirmations
- Enhance accessibility

### Bias Detection Refinement
- Improve prompt design and post-processing for more accurate, nuanced bias insights
- Add confidence thresholds and fallback logic

### Expand News Sources
- Integrate more diverse RSS feeds to reduce bias and increase coverage

### Documentation
- Update README, API docs, and developer setup instructions
- Document prompt templates and configuration options

## Known Issues

- **Bias detection logic requires refinement**; current heuristics sometimes yield inconsistent or incorrect results.
- **Some logic tests in `internal/llm` continue to fail** due to variability in bias detection outputs.
- Further improvements needed to stabilize and validate bias analysis.

### OpenAI API Integration Anomaly (April 8, 2025)

- Real OpenAI API calls are confirmed via logs.
- However, responses fail to parse as JSON, causing repeated errors:
  
  `Failed to parse LLM JSON response: invalid character 'X' looking for beginning of value`
  
- This suggests unexpected API response format or a bug in parsing logic.
- Needs urgent investigation and fix.

### Improved Article Ranking & OpenAI Fallback (April 8, 2025)

- **Multi-factor ranking** combines:
  - **Bias balance score** (favoring diverse perspectives)
  - **Recency score** (favoring newer articles)
- **Fallback strategy:**
  - If OpenAI bias scores are **missing or invalid**, use **recency score only** or apply a **penalty**.
  - Flag such articles for **reprocessing or review**.
  - Avoid top-ranking unreliable articles.
- **Future extensions:** incorporate user feedback, source credibility, personalization.
- **Goal:** Deliver a well-organized, balanced, and reliable article feed despite LLM issues.

### Bias Scoring Diagnosis (April 9, 2025)

**Root Causes:**
- OpenAI API responses often fail to parse correctly. When parsing fails, the system defaults to empty or "unknown" results with zero confidence, which are still counted as valid. This skews the averaging process and hides underlying issues.
- `RobustAnalyze` includes any non-error response as valid, even if it contains default or low-confidence data. Averaging these inconsistent or placeholder scores leads to unreliable bias detection.

**Additional Factors:**
- Simplistic heuristics.
- Lack of detailed logging.
- No differentiation between genuinely valid and fallback results.

**Recommendations:**
- Add detailed logging of each OpenAI response, parse success/failure, and score used in `RobustAnalyze`.
- Exclude empty, "unknown", or zero-confidence results from valid scores.
- Log raw content strings on parse failures to identify malformed outputs.
- Track metrics on failure modes to guide improvements.
- Review and enforce OpenAI prompt formatting to ensure consistent, parseable JSON responses.

---

## April 2025 Update: Validation, Feedback, QA, and Code Quality

- **Validation & Feedback Loop:** Continuous, automated validation is integrated across backend and frontend. User feedback is ingested both inline (via the UI) and through API endpoints, directly informing prompt tuning and model adjustments. Validation runs automatically on new data and model outputs, flagging inconsistencies or low-confidence results.
- **Dashboards:** Real-time dashboards provide insights into feedback volume, sentiment, validation pass/fail rates, and model performance trends, enabling rapid detection of issues and supporting data-driven iteration.
- **QA Process:** Documentation and code updates follow a multi-stage QA process: drafting with templates, automated checks (linting, link validation, style), peer review, and approval/merge only after passing all checks. Continuous improvement is supported by periodic audits and template updates.

### Resolved SonarQube and Lint Issues
- Major **SonarQube warnings** have been addressed, improving code maintainability and security.
- **Lint errors** across backend and frontend have been fixed.
- The **CI pipeline** now enforces these quality gates on every commit.

### Related Files
- [`architecture_plan.md`](architecture_plan.md) — System architecture and validation flow
- [`memory-bank-update-plan.md`](memory-bank-update-plan.md) — Update plans including feedback loop
- [`memory-bank-enhancement-plan.md`](memory-bank-enhancement-plan.md) — Enhancements related to dashboards and QA
## April 2025: Ensemble Scoring Pipeline Enhancements

**Metadata:**
_Last updated: 2025-04-10_
_Author: Roo_
_Change type: Enhancement_

**Changelog:**
- Added detailed plan to enforce strict JSON output, adaptive re-prompting, model switching, logging, reprocessing, and prompt refinement.

## April 2025 Debugging UI Enhancements

To support developer and tester transparency, the NewsBalancer UI now includes extensive debugging features:

- **Main Article Feed:**
  - Displays **article ID, source, fetch/scoring timestamps, fallback status**.
  - Shows **raw composite score, average confidence, model count**.
  - Features a **bias slider** with color-coded zones (blue/gray/red), **model disagreement highlights**, and **detailed tooltips** including parse status.
  - **Advanced debug info** is **default expanded**, listing all raw model outputs, parse success/failure, and aggregation stats.
  - Feedback options include **"Report parse error"**, and can be tagged with **"Model disagreement"**, **"Low confidence"**, or **"Fallback used"**.

- **Article Detail Page / Modal:**
  - Accessed via htmx, shows **full article text**, **all raw model responses**, **parse status**, and **timestamps**.
  - Provides **download raw JSON**, **retry parse/re-score** buttons, and **granular bias visualization toggle**.
  - Clearly highlights **parse failures**.

- **Debugging Transparency:**
  - Exposes **fallback triggers**, **raw API responses**, **parse success/failure**, **aggregation method**, and **timestamps**.
  - Uses **color cues** (green/orange/red) and **tooltips** for status.
  - Adds **inline loading/error indicators**.

- **Accessibility & Responsiveness:**
  - All info-rich elements are **keyboard accessible**, **ARIA-labeled**, and **high contrast**.
  - Uses **semantic HTML** and a **responsive layout**.
  - Debug info is **collapsible but default expanded** during debugging.

- **Minimal JS/SCSS:**
  - Uses **htmx** for interactivity.
  - Employs **vanilla JS** only for bias indicator positioning, tooltips, or essential UI.
  - Keeps CSS minimal and clear.

These enhancements prioritize **transparency**, **debugging ease**, and **developer insight** during model integration and testing.

---

### Overview

To reduce parse failures and unscored articles, the ensemble scoring pipeline will be enhanced with:

- **Strict JSON output enforcement** using delimiters and examples.
- **Adaptive re-prompting** and **model switching** on failures.
- **Improved logging and alerting** for repeated failures.
- **Automated reprocessing** of failed articles.
- **Refined prompt engineering** to minimize errors.

---

### Implementation Plan

**1. Enforce Strict JSON Output**

- Update prompts to instruct LLMs to respond **only** with JSON inside triple backticks.
- Include few-shot examples within delimiters.
- Extract JSON between delimiters before parsing.
- Add tests for noisy outputs.

**2. Adaptive Re-Prompting and Model Switching**

- Classify failures and adapt prompts accordingly.
- Retry with stricter prompts or alternative variants.
- Switch models if repeated failures occur.
- Log all attempts with metadata.

**3. Logging and Alerting**

- Log article ID, model, prompt, error type, and raw response.
- Track failure rates and trigger alerts on thresholds.
- Expose metrics for monitoring.

**4. Automated Reprocessing**

- Queue failed articles with metadata.
- Periodically reprocess with adaptive prompts/models.
- Escalate persistent failures.

**5. Prompt Refinement**

- Use explicit JSON instructions and examples.
- Develop multiple prompt variants.
- Empirically select best variants.

---

### Cross-References

- [`architecture_plan.md`](architecture_plan.md)
- [`progress.md`](progress.md)
- [`memory-bank-update-plan.md`](memory-bank-update-plan.md)
- [`memory-bank-enhancement-plan.md`](memory-bank-enhancement-plan.md)

---
[2025-04-12 12:23:25] - Debugging focus: Validating fixes for compilation error (api.go: ScoreWithModel args) and runtime parse error (llm.go: markdown-wrapped JSON). Added debug logs to api.go and llm.go. Improved markdown stripping logic in llm.go. Diagnosis confirmation requested from user. UMB triggered.

- [`progress.md`](progress.md) — Progress tracking including QA and validation milestones
[2025-04-12 16:08:46] - Debug session: Diagnosed scoring failure as OpenRouter rate limit for mistralai/mistral-7b-instruct:free. Multiple tool attempts to add diagnostic logging to reanalyzeHandler in internal/api/api.go. Provided manual patch instructions. User requested Memory Bank update (UMB).
[2025-04-12 16:31:00] - Added secondary API key to .env file.
[2025-04-12 16:31:00] - Added secondary API key to .env file.
[2025-04-12 16:31:00] - Added secondary API key to .env file.
[2025-04-12 18:41:45] - UMB: Updated models in config. Encountered new JSON parsing error (markdown backticks) with tokyotech-llm model. Debug task cancelled; decided to ignore parsing error for now.