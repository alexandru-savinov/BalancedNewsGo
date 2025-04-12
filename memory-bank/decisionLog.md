# Decision Log

This file records architectural and implementation decisions for the NewsBalancer project. Each entry includes a timestamp, summary, rationale, and implementation details.

---

[2025-04-12 12:41:40] - Adopted multi-model, multi-prompt ensemble for bias detection

## Decision

Implemented a robust ensemble architecture using multiple LLMs (OpenAI, OpenRouter, Claude, etc.) and diverse prompt templates to assess article bias from multiple perspectives.

## Rationale

Single-model approaches were insufficiently robust and prone to individual model biases. An ensemble increases reliability, diversity of perspectives, and reduces the risk of systematic errors.

## Implementation Details

Integrated multiple LLM providers and prompt variants. Aggregated results using mean and variance, with outlier filtering and confidence scoring. Modularized the backend to support easy addition of new models and prompts.

---

[2025-04-12 12:41:40] - Transitioned from discrete bias labels to continuous bias scoring

## Decision

Replaced categorical bias labels (Left, Center, Right) with a continuous bias score ranging from -1.0 (strongly left) to 1.0 (strongly right), in 0.1 increments.

## Rationale

Continuous scoring enables more nuanced, fine-grained analysis and supports advanced analytics, visualization, and feedback loops.

## Implementation Details

Prompt engineering was updated to request explicit numerical scores. The database schema and API were extended to store and expose continuous scores. Frontend visualizations (sliders, histograms) were added.

---

[2025-04-12 12:41:40] - Integrated continuous validation and feedback loop

## Decision

Established a continuous validation and feedback loop, ingesting user feedback and flagged outliers to refine prompts, models, and aggregation logic.

## Rationale

Ongoing validation and user feedback are essential for improving model accuracy, detecting edge cases, and adapting to changing news and model behaviors.

## Implementation Details

API endpoints and frontend forms collect user feedback. Automated validation runs on new data and model outputs, flagging inconsistencies or low-confidence results. Dashboards provide real-time insights.

---

[2025-04-12 12:41:40] - Chose OpenAI API (and later OpenRouter) as primary LLM provider

## Decision

Initially integrated OpenAI API for LLM-based bias detection, later migrating to OpenRouter for expanded model access and cost optimization.

## Rationale

OpenAI provided reliable, high-quality LLMs. OpenRouter enabled access to additional models and improved cost control.

## Implementation Details

Backend LLM integration was abstracted to support multiple providers. Migration involved updating API clients, prompt templates, and error handling.

---

[2025-04-12 12:41:40] - Adopted htmx for dynamic UI updates

## Decision

Used htmx to enable dynamic, partial page updates in the frontend, improving responsiveness and user experience with minimal JavaScript.

## Rationale

htmx allows for interactive UI features (e.g., feedback forms, bias sliders) without the complexity of a full SPA framework, supporting rapid iteration and maintainability.

## Implementation Details

Frontend templates were refactored to use htmx attributes for dynamic content loading and inline feedback submission. Minimal vanilla JS was retained for advanced UI elements.

---

[2025-04-12 12:41:40] - Selected SQLite for initial data storage

## Decision

Chose SQLite as the initial database for storing articles, LLM results, and user feedback.

## Rationale

SQLite is lightweight, easy to set up, and sufficient for MVP and early-stage deployments. It supports rapid prototyping and local development.

## Implementation Details

Database schema was designed for multi-perspective scores and feedback. Repository pattern abstracts DB access for future migration to other databases.

---

[2025-04-12 12:41:40] - Implemented verbose logging and diagnostics

## Decision

Introduced verbose, structured logging across backend modules to facilitate debugging, monitoring, and quality control.

## Rationale

Detailed logs are essential for diagnosing LLM integration issues, tracking API failures, and supporting continuous improvement.

## Implementation Details

Logging was added to all critical backend components, including LLM API calls, parsing, aggregation, and error handling. Logs are structured for easy analysis and monitoring.

---

[2025-04-12 12:41:40] - Enforced code quality with golangci-lint and CI pipeline

## Decision

Integrated golangci-lint and SonarQube into the CI pipeline to enforce code quality, security, and maintainability.

## Rationale

Automated linting and static analysis prevent regressions, improve code health, and support long-term maintainability.

## Implementation Details

CI/CD pipeline runs golangci-lint and SonarQube checks on every commit. Major warnings and lint errors were resolved as part of the 2025 redesign.


---

[2025-04-12 18:41:45] - Updated LLM models for bias perspectives

## Decision

Updated the models used for left, center, and right perspectives in `configs/composite_score_config.json` to:
- Left: `tokyotech-llm/llama-3.1-swallow-8b-instruct-v0.3`
- Center: `mistralai/mistral-small-3.1-24b-instruct`
- Right: `google/gemini-2.0-flash-001`

## Rationale

Attempting to use models without rate limits to resolve scoring issues.

## Implementation Details

Modified the `modelName` fields in the `models` array within `configs/composite_score_config.json`.


---

[2025-04-12 18:41:45] - Updated LLM models for bias perspectives

## Decision

Updated the models used for left, center, and right perspectives in `configs/composite_score_config.json` to:
- Left: `tokyotech-llm/llama-3.1-swallow-8b-instruct-v0.3`
- Center: `mistralai/mistral-small-3.1-24b-instruct`
- Right: `google/gemini-2.0-flash-001`

## Rationale

Attempting to use models without rate limits to resolve scoring issues.

## Implementation Details

Modified the `modelName` fields in the `models` array within `configs/composite_score_config.json`.