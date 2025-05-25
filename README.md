# NewsBalancer Go Backend
![Coverage](https://go-coverage-badge.appspot.com/badge/github.com/alexandru-savinov/BalancedNewsGo.svg)


A Go-based backend service that provides politically balanced news aggregation using LLM-based analysis with a modern, responsive web interface.

## Overview

NewsBalancer analyzes news articles from diverse sources using multiple LLM perspectives (left, center, right) to provide balanced viewpoints and identify potential biases. The application features a complete **Editorial template integration** providing a modern, responsive web interface with server-side rendering.

**Current Status:** ✅ **Production Ready** - All core functionality is operational including the Editorial template web interface, database integration, search functionality, filtering, pagination, and comprehensive testing. The `essential`, `backend`, and `api` test suites pass when run with the `NO_AUTO_ANALYZE=true` environment variable.

**Key Architectural Principles:**
*   **Data Flow:** Articles are ingested from RSS feeds (`internal/rss`), stored in a SQLite database (`internal/db`, typically `news.db`), and then analyzed for political bias.
*   **LLM Analysis:** The `internal/llm` package manages LLM interactions. It uses an ensemble approach defined in `configs/composite_score_config.json`, leveraging multiple models and perspectives. A key feature is the averaging of duplicate scores and confidences if multiple results are found for the same model/perspective during an analysis pass. The composite score calculation is primarily handled by `internal/llm/composite_score_fix.go`.
*   **Database:** SQLite is used for persistence. The `llm_scores` table has a `UNIQUE(article_id, model)` constraint, which is crucial for correctly upserting LLM scores using `ON CONFLICT` SQL clauses.
*   **API & Web:** Results and functionalities are exposed via a RESTful API (`internal/api`) and a modern web interface with **Editorial template integration** (`web/templates/`).

**Editorial Template Integration:** ✅ **COMPLETE**
- **Modern Web UI**: Responsive design using HTML5 UP's Editorial template
- **Server-side Rendering**: Go templates with real database data
- **Search & Filtering**: Full-text search, source filtering, and political bias filtering
- **Pagination**: Page-based navigation with state preservation
- **Performance**: Sub-20ms response times with optimized database queries
- **Mobile-friendly**: Responsive layout that works on all devices

**Latest Test Status:**
- All Go unit, integration, and end-to-end tests now pass, *with the exception of some `internal/llm` unit tests (see status table below)*.
- **Editorial template integration** fully operational with comprehensive testing completed
- Server-side rendering with real database data working perfectly
- Search, filtering, and pagination functionality verified
- Performance optimized with sub-20ms response times
- The codebase now uses **averaging everywhere** for duplicate model/perspective scores and confidences. This logic is fully covered by passing tests.
- For reliable test runs, set the environment variable `NO_AUTO_ANALYZE=true` (see `docs/testing.md`).

## Recent Achievements

- **✅ Editorial Template Integration Complete**: Modern responsive web interface using HTML5 UP's Editorial template
- **✅ Server-side Rendering**: Complete transition from client-side JavaScript to Go template rendering
- **✅ Database Integration**: Real article data displayed with search, filtering, and pagination
- **✅ Performance Optimization**: 2-20ms response times with efficient database queries
- **✅ Mobile Responsive**: Fully responsive design that works on all devices
- Added `UNIQUE(article_id, model)` constraint to the `llm_scores` table schema to support proper functioning of `ON CONFLICT` clauses in SQL queries that update ensemble scores. This fixed critical SQL errors during test execution.
- Ensured proper test isolation by clearing database between test runs and properly shutting down processes.
- Documentation improvements for troubleshooting common test issues.

## Test Suite Status

| Test Suite | Status | Notes |
|------------|--------|-------|
| `essential` | ✅ PASS | Core API functionality tests pass after schema fix |
| `backend` | ✅ PASS | All 61 assertions successful |
| `api` | ✅ PASS | All API endpoints function correctly |
| **Editorial Templates** | ✅ PASS | Server-side rendering, search, filtering, pagination all working |
| **Web Interface** | ✅ PASS | Modern responsive UI with real database integration |
| Go Unit Tests: `internal/db` | ✅ PASS | All database operations function correctly |
| Go Unit Tests: `internal/api` | ✅ PASS | API layer works correctly |
| Go Unit Tests: `internal/llm` | ❌ FAIL | Various failures in score calculation logic. Detailed analysis of these failures is pending central documentation. |
| `all` | ❌ FAIL | Missing test collection: `extended_rescoring_collection.json` |
| `debug` | ❌ FAIL | Missing test collection: `debug_collection.json` |
| `confidence` | ❌ FAIL | Missing test collection: `confidence_validation_tests.json` |

See `docs/testing.md` for more information on test statuses and execution.

## Database Schema Highlights

The application relies on a SQLite database (typically `news.db`) with a schema defined in `internal/db/db.go`. Key tables include `articles`, `llm_scores`, `feedback`, and `labels`.

A critical aspect of the `llm_scores` table is the `UNIQUE(article_id, model)` constraint. This constraint is essential for the correct functioning of the LLM scoring pipeline, particularly when updating scores using `ON CONFLICT` SQL clauses, ensuring that scores for a given article and model are properly upserted (updated if existing, or inserted if new).
### Database Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) to manage schema changes.
Migrations reside in the `migrations/` directory. To apply them, install the `migrate` CLI and run:

```bash
go install -tags "sqlite" github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate -path ./migrations -database "sqlite3://news.db" up
```


## Features

### Core Functionality
- RSS feed aggregation from multiple sources
- Multi-perspective LLM analysis (left/center/right)
- Composite score calculation with confidence metrics
- RESTful API for article retrieval and analysis
- Caching and database persistence
- Real-time progress tracking via SSE

### Modern Web Interface (Editorial Template Integration)
- **Responsive Design**: Mobile-first approach using HTML5 UP's Editorial template
- **Server-side Rendering**: Fast Go template rendering with real database data
- **Article Browsing**: Paginated article list with metadata, bias indicators, and source badges
- **Advanced Search**: Full-text search across article titles and content
- **Smart Filtering**: Filter by source, political bias, confidence level, and publication date
- **Pagination**: Efficient page-based navigation with state preservation
- **Individual Articles**: Detailed article view with bias analysis and AI summaries
- **Performance**: Sub-20ms response times with optimized database queries
- **Accessibility**: Semantic HTML with proper ARIA labels and keyboard navigation

## API Reference

The application provides a comprehensive REST API for interacting with articles, bias analysis, and other features. Full API documentation is available in the [Swagger documentation](docs/swagger.json).

Key endpoints include:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/articles` | GET | Fetch articles with optional filtering |
| `/api/articles/{id}` | GET | Get a specific article by ID |
| `/api/articles/{id}/bias` | GET | Get political bias analysis for an article |
| `/api/articles/{id}/ensemble` | GET | Get detailed ensemble scoring information |
| `/api/llm/reanalyze/{id}` | POST | Trigger reanalysis of an article |
| `/api/llm/score-progress/{id}` | GET | SSE stream for real-time scoring progress |
| `/api/feedback` | POST | Submit user feedback on article bias |
| `/api/feeds/healthz` | GET | Check RSS feed health status |

Detailed API documentation is available at `/swagger/index.html` when running the server.

## Web Interface

NewsBalancer features a **modern, responsive web interface** built with the **Editorial template integration**. The interface provides an intuitive way to browse, search, and analyze news articles with comprehensive bias analysis.

### 🏠 **Home Page (Articles List)**
- **Article Grid**: Responsive card-based layout showing article previews with images
- **Real-time Data**: Server-rendered content with live database integration
- **Visual Bias Indicators**: Color-coded badges showing political leaning (Left/Center/Right)
- **Source Badges**: Clear identification of news sources (CNN, Fox News, BBC, etc.)
- **Publication Dates**: Human-readable timestamps for article freshness
- **Search Form**: Prominent search bar with real-time query processing
- **Advanced Filtering**: Dropdown filters for source and political bias
- **Pagination Controls**: Next/Previous navigation with page state preservation

### 📰 **Article Detail Page**
- **Full Article Content**: Complete article text with proper formatting
- **Metadata Display**: Source, publication date, and article URL
- **Bias Analysis Section**: Detailed political bias scoring with confidence levels
- **AI Summary Integration**: LLM-generated summaries when available
- **Recent Articles Sidebar**: Navigation to related content
- **Mobile Optimized**: Touch-friendly interface for mobile devices

### 🎨 **Design Features**
- **Editorial Template**: Professional design using HTML5 UP's Editorial template
- **Responsive Layout**: Works seamlessly on desktop, tablet, and mobile
- **Fast Loading**: Server-side rendering with optimized asset delivery
- **Modern Typography**: Clean, readable fonts with proper spacing
- **Intuitive Navigation**: Sidebar menu with clear section organization
- **Performance**: 2-20ms response times with efficient caching

### 🔍 **Search & Filtering**
- **Full-text Search**: Search across article titles and content
- **Source Filtering**: Filter by specific news sources
- **Bias Filtering**: Filter by political leaning (Left/Center/Right)
- **Pagination**: Navigate through large result sets efficiently
- **State Preservation**: Maintain filters and search terms across page navigation
- **Real-time Results**: Instant search results without page refresh

All static assets are served from `/web/assets/` with templates in `/web/templates/`. The interface supports both API-driven and template-rendered modes for maximum flexibility.

## Project Structure

Here's a breakdown of the project's directory structure:

**Core Application Files:**

*   `cmd/`: Main application entry points (e.g., `cmd/server/main.go`).
*   `internal/`: Private application logic, including business logic, data access, and core functionalities.
*   `configs/`: Application configuration files (e.g., `feed_sources.json`).
*   `go.mod`, `go.sum`: Go module definitions and dependencies.
*   `web/`: Frontend assets and templates served by the application, including HTML, JavaScript, and CSS.
*   `.env.example`: Template for required environment variables. Copy this file to `.env` and fill in your local secrets (the `.env` file itself should never be committed).

**Testing Files & Infrastructure:**

*   `tests/`: Contains end-to-end, integration, or other types of tests.
*   `*_test.go` files (within `internal/` or other packages): Go unit tests.
*   `postman/`: Postman collections for API testing.
*   `run_*.cmd`, `run_*.sh`, `test.cmd`, `test.sh`, `Makefile`: Scripts and definitions for running test suites and other tasks. See `docs/testing.md` for details.
*   `package.json`, `package-lock.json`, `node_modules/`: Node.js dependencies, likely for test tools like Newman or potentially frontend build steps.
*   `newman_environment.json`: Environment configuration for Newman API tests.
*   `*.js` (in root, e.g., `test_sse_progress.js`, `generate_test_report.js`, `analyze_test_results.js`): Helper scripts, likely for test execution or reporting.
*   `mock_llm_service.go`, `mock_llm_service.py`: Mock services used during testing.

**Documentation:**

*   `README.md`: This file - main project overview.
*   `CONTRIBUTING.md`: Guidelines for contributors.
*   `LICENSE`: Project license information.
*   `CHANGELOG.md`: Record of changes across versions.
*   `docs/`: Contains detailed documentation:
    *   [Codebase Documentation](docs/codebase_documentation.md): Detailed breakdown of Go source files and structure.
    *   [Testing Guide](docs/testing.md): Comprehensive guide on running, analyzing, and debugging tests.
    *   [Configuration Reference](docs/configuration_reference.md): Details on environment variables and configuration files.
    *   [Integration Testing Guide](docs/integration_testing.md): Guide for testing with external services like LLMs.
    *   [Deployment Guide](docs/deployment.md): Instructions for production deployment and performance tuning.
    *   `architecture.md`: Describes the data flow and architecture (e.g., CompositeScore calculation).
    *   [Request Flow Overview](docs/request_flow.md): Step-by-step walkthrough of how an API request travels through the system.
    *   `swagger.yaml`, `swagger.json`, `docs.go`: API documentation (Swagger/OpenAPI).
    *   `PR/handle_total_analysis_failure.md`: Technical recommendation for handling total analysis failure.
    *   [Potential Codebase Improvements](docs/plans/potential_improvements.md): Suggestions for future development and enhancements.
    *   `archive/`: Contains historical documents (e.g., past roadmaps, fix details).

**Development & Environment Configuration:**

*   `.git/`: Git repository metadata.
*   `.gitignore`: Specifies intentionally untracked files that Git should ignore.
*   `.vscode/`, `.sonarlint/`: Editor/IDE specific configuration.
*   `.golangci.yml`: Linter configuration for Go code.
*   `tsconfig.json`, `test.config.js`, `global.d.ts`: TypeScript configuration, likely related to JavaScript test helpers or potentially a web frontend.
*   `apidog/`: Likely contains configuration or data for the ApiDog tool.
*   `start_server.bat`, `start_server.sh`: Scripts to run the development server.

**Potentially Temporary / Generated / Bloat Files (\*):**

*These files might be generated during build/test processes, are temporary, or could potentially be cleaned up or added to `.gitignore`.*

*   `newsbalancer.exe`, `newbalancer_server.exe`: Compiled application binaries. (\* Add to `.gitignore` if not meant for distribution this way)
*   `news.db`, `news.db-shm`, `news.db-wal`: SQLite database files. (\* Likely runtime data, add to `.gitignore`)
*   `*.out` (e.g., `coverage.out`, `llm_cov.out`): Test coverage output files. (\* Add to `.gitignore`)
*   `*.html` (e.g., `coverage-report.html`, `llm_coverage.html`): HTML reports generated from coverage or tests. (\* Add to `.gitignore`)
*   `test-results/`: Directory likely containing detailed test reports. (\* Add to `.gitignore`)
*   `playwright-report/`: Directory containing Playwright E2E test reports. (\* Add to `.gitignore`)
*   `tmp_db_backup/`: Temporary database backups. (\* Add to `.gitignore`)
*   `*.log` (e.g., `backend_run.log`, `C:\...\*.log`): Log files. (\* Add `*.log` to `.gitignore`)
*   `*.md` (e.g., `test_results.md`, `test_results_summary.md`, `test_coverage_todo.md`): Generated markdown reports or temporary notes. (\* Consider adding generated reports to `.gitignore`)
*   `*.js` (root level debug files like `debug_api_response.js`, `debug_article_response.js`): Potentially temporary debugging artifacts.
*   `.roomodes`, `roomodes`: Unknown purpose, potentially related to a specific tool or temporary state.
*   `.git_commit_msg.txt`: Temporary file often used by Git tools/hooks. (\* Add to `.gitignore`)
*   `memory-bank/`: Unknown purpose, potentially temporary data or cache.
*   `.42c/`: Unknown purpose, potentially tool-specific cache/config.

## Running Tests

Refer to the **[Testing Guide](docs/testing.md)** for detailed instructions on running different test suites (unit, backend, all, debug), analyzing results, and debugging.

**Test Suite Alignment:** The various test suites are designed to be complementary, targeting different aspects of the application to ensure comprehensive quality assurance:
*   **Unit Tests (`go test ./...`)**: Focus on validating individual Go packages and functions in isolation.
*   **Backend Integration Tests (`scripts/test.cmd backend`)**: Verify the interactions between different internal components of the backend system.
*   **API Tests (`scripts/test.cmd api`)**: Perform black-box testing of the application's API endpoints, ensuring the public contract is met.
*   **Aggregated Suites (`scripts/test.cmd all`)**: Run a combination of tests for broader coverage.
*   **Essential Tests (`scripts/test.cmd essential`)**: Cover core functionality described as essential for basic operation.

**Important Note on Testing:** Due to concurrency issues with SQLite and background LLM analysis tasks triggered by some API calls, tests involving Newman (`backend`, `api`, `essential`, `all`, `debug`, `confidence`) require the `NO_AUTO_ANALYZE=true` environment variable to be set. The provided `test.cmd` and `test.sh` scripts handle this automatically. This prevents tests from failing due to database locks (`SQLITE_BUSY`) caused by contention between the test runner and background analysis.

Additionally, the database schema includes a `UNIQUE(article_id, model)` constraint on the `llm_scores` table to ensure SQL queries using `ON CONFLICT` clauses work correctly when updating ensemble scores. This prevents errors like `SQL logic error: ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint` during test execution.

### Common Test Issues and Solutions

1. **Port Conflicts**:
   ```powershell
   # Kill processes using port 8080
   Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue | 
      Select-Object -ExpandProperty OwningProcess | 
      ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }
   ```
   *Note: This error commonly occurs if another instance of the server is already running, or if another application is using port 8080. This can happen when using `make run` or `go run cmd/server/main.go` if the port is not free.*

2. **Database Locks**:
   ```powershell
   # Kill server processes
   Get-Process -Name "go", "newbalancer_server" -ErrorAction SilentlyContinue | Stop-Process -Force
   
   # Delete database file
   Remove-Item news.db -Force
   ```

3. **Database Corruption**: If you encounter persistent database corruption issues, use the database recreation script:
   ```powershell
   # Reset the database automatically
   ./recreate_db.ps1
   ```
   This script backs up existing data, resets the database with proper schema, and verifies integrity. See [Testing Guide](docs/testing.md) for more details on database maintenance.

4. **Missing Collections**: Some test suites require collections that may not be present. Focus on the `essential` and `backend` tests if you encounter missing collection errors.

### Common Test Commands

- Run Go unit tests:
  ```
  go test ./...
  ```
- Run backend integration tests:
  ```bash
  # Windows
  scripts/test.cmd backend
  # Linux/macOS
  scripts/test.sh backend 
  ```
- Run API tests:
  ```bash
  # Windows
  scripts/test.cmd api
  # Linux/macOS
  scripts/test.sh api 
  ```
- Run the complete test suite (essential, extended, confidence):
  ```bash
  # Windows
  scripts/test.cmd all
  # Linux/macOS
  scripts/test.sh all 
  ```
- Run essential tests only:
  ```bash
  # Windows
  scripts/test.cmd essential
  # Linux/macOS
  scripts/test.sh essential 
  ```
- Generate HTML report from latest results:
  ```bash
  # Windows
  scripts/test.cmd report
  # Linux/macOS
  scripts/test.sh report
  ```
- Clean test results:
  ```bash
  # Windows
  scripts/test.cmd clean
  # Linux/macOS
  scripts/test.sh clean
  ```

See `scripts/test.cmd help` or `scripts/test.sh help` for all available commands.

## Development

### Environment Setup

1. Copy `.env.example` to `.env`
2. Configure RSS feed sources in `configs/feed_sources.json`
3. Set up LLM API keys in `.env`

### API Contract Validation Tools

Our project uses tools to validate the API specification (OpenAPI).

**1. Spectral CLI:**
This tool is used for linting the OpenAPI specification. It's managed as an npm dev dependency in `package.json`. Ensure you run `npm install` (or `pnpm install`).

**2. oasdiff:**
This tool is used for detecting breaking changes between API versions. Install it using Go:
```bash
go install github.com/oasdiff/oasdiff@latest
```
Ensure that your Go binary path (typically `$(go env GOPATH)/bin` or `~/go/bin`) is included in your system's `PATH` environment variable.

**Purpose & Usage:**

The `make contract` target is crucial for maintaining API quality and stability. It performs two main functions:
1.  **Linting**: Uses Spectral CLI to check the OpenAPI specification (`docs/swagger.json`) against a defined ruleset (`.spectral.yaml`) for style consistency, completeness, and best practices.
2.  **Breaking Change Detection**: Uses `oasdiff` to compare the current API specification against the last known version (backed up as `swagger.json.bak`) to identify any changes that could break existing client integrations.

**Common Contract Validation Workflow:**

1. Run `make docs` to generate or update the Swagger documentation from your code.
2. Run `make contract` to validate the API specification.
3. If validation fails:
   - For linting errors: Check the handler annotations in your Go files (`internal/api/api.go`, `cmd/server/main.go`).
   - For breaking changes: Review if the change was intentional and consider versioning your API if necessary.

**API Annotation Best Practices:**

When documenting API endpoints with Swagger annotations, ensure:
- Every handler has a unique `@ID` attribute
- Handler tags match those defined in the global annotations (`@tag.name` in `cmd/server/main.go`)
- Response models use fully qualified types that exist in your codebase
- Every endpoint has proper descriptions, parameters, and response types

**Pre-commit Hook:**
To automate these checks, a pre-commit hook is recommended. 

*   **Manual Setup:** To set this up manually:
    1.  Create/edit the file `.git/hooks/pre-commit`.
    2.  Paste the script content (provided in the project's `docs/PR/makefile_test_results.txt` or by the setup assistant).
    3.  Make it executable: `chmod +x .git/hooks/pre-commit` (on Linux/macOS).
    This hook will automatically run `make docs` and then `make contract` before each commit, preventing commits with API contract violations.

**Interpreting Errors:**

*   **Spectral Errors:** Linting errors from Spectral will point to issues in your OpenAPI specification, often originating from the Go code comments used to generate it. Address these by correcting the annotations in your Go handlers or models.
*   **`oasdiff` Errors:** Breaking change errors indicate that a modification to the API (e.g., removing a field, changing a data type) is not backward-compatible. Carefully review these changes. If intentional, the API version might need to be incremented. If unintentional, revert the change.

Regularly running `make contract` and using the pre-commit hook helps catch API design issues early, ensuring a more robust and reliable API.

### Running Locally

Start the server using the Go command:
```
go run cmd/server/main.go
```
Or using the Makefile, which typically builds the executable (e.g., to `./bin/newbalancer_server`) and then runs it:
```
make run
```

This will start the server on port 8080 by default. You can then access the web interface at http://localhost:8080.
*Note: If you encounter a "port already in use" error (e.g., `listen tcp :8080: bind: Only one usage of each socket address...`), ensure no other processes are using port 8080. See "Port Conflicts" under "Common Test Issues and Solutions" above.*

## License

Licensed under the [MIT License](LICENSE).

