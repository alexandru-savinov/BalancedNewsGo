<!-- Metadata -->
Last updated: April 10, 2025  
Author: Roo AI Assistant  

# Changelog
- **2025-04-10:** Merged `systemPatterns.md` and `techContext.md` into this unified `systemTech.md` file. Updated metadata and changelog accordingly.
- **2025-04-09:** Previous updates to `systemPatterns.md` and `techContext.md`.

---

# System Architecture, Patterns, and Technology Context

---

## System Patterns (from `systemPatterns.md`)

### Architecture Overview
- Modular Go backend with clear separation of concerns:
  - **RSS Module (`internal/rss`)**: Fetches and parses feeds from multiple sources
  - **Database Module (`internal/db`)**: Handles SQLite persistence for articles and LLM outputs
  - **LLM Module (`internal/llm`)**: Interfaces with OpenAI API or local models, abstracts provider differences
  - **API Module (`internal/api`)**: Exposes REST endpoints for frontend consumption
- SQLite database for lightweight, local storage
- Integration with LLMs (OpenAI API or local models) via adapter pattern
- REST API layer serving JSON
- Frontend using HTML, JavaScript, and htmx for interactivity

### Backend Data Flow
1. **Fetch**: RSS module retrieves articles from diverse news sources
2. **Store Raw**: Raw articles saved in SQLite via database module
3. **Analyze**: LLM module processes articles to generate summaries, extract perspectives, and identify potential bias
4. **Store Analysis**: Summaries and metadata linked to original articles in database
5. **Serve**: API module exposes endpoints to retrieve articles, summaries, perspectives
6. **Display**: Frontend fetches and dynamically displays balanced news content

### Key Design Patterns
- **Repository Pattern**: Abstracts database access
- **Adapter Pattern**: Supports multiple LLM providers seamlessly
- **RESTful API**: Clean separation between backend and frontend
- **Progressive Enhancement**: htmx augments static HTML with dynamic updates

### Component Relationships
- RSS module feeds data into the database
- LLM module reads raw articles from database, writes back analysis
- API module queries database for both raw and analyzed data
- Frontend interacts solely with API module

### Notes from Backend Audit
- Modular design confirmed effective
- Adapter pattern for LLMs simplifies provider switching
- Data flow is clear and maintainable
- Opportunities:
  - Improve LLM prompt engineering for richer multi-perspective output
  - Expand API endpoints (e.g., user feedback, comparison views)
  - Enhance bias detection logic in LLM processing

---

## Technology Context (from `techContext.md`)

### Technology Stack
- **Backend:** Go (Golang)
- **Database:** SQLite (local, lightweight)
- **Language Models:** OpenAI GPT APIs or local LLMs
- **Frontend:** HTML, JavaScript, htmx for dynamic UI
- **API:** RESTful, JSON-based

### Development Setup
- Go modules for dependency management
- SQLite database file stored locally
- LLM API keys configured via environment variables or config files
- Frontend served via Go HTTP server

### Technical Constraints
- Lightweight and local-first design
- Minimize LLM API costs and latency
- Modular to support multiple LLM providers
- Extendable for new sources and features

### Dependencies
- Go standard library
- SQLite driver for Go
- LLM API clients (OpenAI, etc.)
- htmx JavaScript library

### Development Tools and Configuration
- Adopted **golangci-lint** with stricter linting rules to enforce code quality and consistency.
- Added a `.golangci.yml` configuration file to customize linting behavior.
- Improved configuration loading: the system now **falls back to environment variables** if configuration files are missing or incomplete, enhancing robustness.
- Enhanced diagnostics by adding **verbose logging** throughout backend components, aiding debugging and monitoring.

---

### Debugging UI & Frontend Approach (April 2025)
- The frontend uses **htmx** extensively for interactivity, dynamic content loading, and modals/pages.
- **Minimal vanilla JavaScript** is used only for bias slider positioning, tooltips, and essential UI logic.
- **Minimal SCSS/CSS**, focusing on clarity, responsiveness, and color cues.
- The UI exposes **fallback triggers, raw API responses, parse success/failure, aggregation methods, and timestamps** to maximize transparency.
- **Color cues** (green/orange/red), **tooltips**, and **inline loading/error indicators** provide immediate debugging feedback.
- **Accessibility** is prioritized with ARIA labels, keyboard navigation, high contrast, and semantic HTML.
- The info-rich, transparent UI accelerates **developer and tester debugging**, enabling rapid diagnosis of model issues, parse failures, and fallback scenarios.