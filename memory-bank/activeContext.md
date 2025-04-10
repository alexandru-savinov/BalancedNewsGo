# Active Context

---

**Summary (April 2025):**  
The 2025 redesign is now live, featuring a robust **multi-model, multi-prompt ensemble** for nuanced bias detection. The **API** has been enhanced with endpoints for user feedback and bias insights, while the **frontend** supports dynamic content loading and inline feedback submission. **Prompt engineering** has been refined with configurable templates and few-shot examples, improving LLM reliability. A **continuous validation and feedback loop** guides ongoing improvements. Major **refactoring** improved modularity and maintainability, resolving key **SonarQube warnings** and stabilizing the codebase.

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

### Fully Operational Validation & Feedback Loop
- The system now features a **continuous, automated validation loop** integrated across backend and frontend.
- **User feedback** is ingested both inline (via the UI) and through API endpoints.
- Validation runs automatically on new data and model outputs, flagging inconsistencies or low-confidence results.
- Feedback directly informs **prompt tuning** and **model adjustments**, creating a **closed-loop improvement cycle**.

### Continuous Feedback Ingestion & Dashboards
- Feedback is continuously collected, stored, and analyzed.
- **Dashboards** provide real-time insights into:
  - Feedback volume and sentiment
  - Validation pass/fail rates
  - Model performance trends
- This enables rapid detection of issues and supports **data-driven iteration**.

### QA Framework for Documentation
- A **multi-stage QA process** governs all documentation updates:
  - **Drafting:** Authors use standardized templates to ensure consistency.
  - **Automated Checks:** Markdown linting, link validation, and style enforcement run on every commit.
  - **Peer Review:** At least one reviewer verifies technical accuracy, clarity, and adherence to guidelines.
  - **Approval & Merge:** Only after passing automated checks and peer review can changes be merged.
- **Continuous Improvement:**
  - Collect feedback on documentation usability.
  - Periodic audits to identify outdated or unclear sections.
  - Update templates and checklists based on lessons learned.

### Resolved SonarQube and Lint Issues
- Major **SonarQube warnings** have been addressed, improving code maintainability and security.
- **Lint errors** across backend and frontend have been fixed.
- The **CI pipeline** now enforces these quality gates on every commit.

### Related Files
- [`architecture_plan.md`](architecture_plan.md) — System architecture and validation flow
- [`memory-bank-update-plan.md`](memory-bank-update-plan.md) — Update plans including feedback loop
- [`memory-bank-enhancement-plan.md`](memory-bank-enhancement-plan.md) — Enhancements related to dashboards and QA
- [`progress.md`](progress.md) — Progress tracking including QA and validation milestones