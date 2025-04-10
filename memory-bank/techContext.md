# Technology Context

## Technology Stack
- **Backend:** Go (Golang)
- **Database:** SQLite (local, lightweight)
- **Language Models:** OpenAI GPT APIs or local LLMs
- **Frontend:** HTML, JavaScript, htmx for dynamic UI
- **API:** RESTful, JSON-based

## Development Setup
- Go modules for dependency management
- SQLite database file stored locally
- LLM API keys configured via environment variables or config files
- Frontend served via Go HTTP server

## Technical Constraints
- Lightweight and local-first design
- Minimize LLM API costs and latency
- Modular to support multiple LLM providers
- Extendable for new sources and features

## Dependencies
- Go standard library
- SQLite driver for Go
- LLM API clients (OpenAI, etc.)
- htmx JavaScript library

## Development Tools and Configuration

- Adopted **golangci-lint** with stricter linting rules to enforce code quality and consistency.
- Added a `.golangci.yml` configuration file to customize linting behavior.
- Improved configuration loading: the system now **falls back to environment variables** if configuration files are missing or incomplete, enhancing robustness.
- Enhanced diagnostics by adding **verbose logging** throughout backend components, aiding debugging and monitoring.

---

## Debugging UI & Frontend Approach (April 2025)

- The frontend uses **htmx** extensively for interactivity, dynamic content loading, and modals/pages.
- **Minimal vanilla JavaScript** is used only for bias slider positioning, tooltips, and essential UI logic.
- **Minimal SCSS/CSS**, focusing on clarity, responsiveness, and color cues.
- The UI exposes **fallback triggers, raw API responses, parse success/failure, aggregation methods, and timestamps** to maximize transparency.
- **Color cues** (green/orange/red), **tooltips**, and **inline loading/error indicators** provide immediate debugging feedback.
- **Accessibility** is prioritized with ARIA labels, keyboard navigation, high contrast, and semantic HTML.
- The info-rich, transparent UI accelerates **developer and tester debugging**, enabling rapid diagnosis of model issues, parse failures, and fallback scenarios.