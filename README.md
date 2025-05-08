# NewsBalancer Go Backend

A Go-based backend service that provides politically balanced news aggregation using LLM-based analysis.

## Overview

NewsBalancer analyzes news articles from diverse sources using multiple LLM perspectives (left, center, right) to provide balanced viewpoints and identify potential biases. 

**Current Status:** Basic functionality is working, and the `essential` test suite passes when run with the `NO_AUTO_ANALYZE=true` environment variable to prevent background LLM processing from interfering with tests due to SQLite concurrency limitations.

## Features

- RSS feed aggregation from multiple sources
- Multi-perspective LLM analysis (left/center/right)
- Composite score calculation with confidence metrics
- RESTful API for article retrieval and analysis
- Caching and database persistence
- Real-time progress tracking via SSE

## Project Structure

Here's a breakdown of the project's directory structure:

**Core Application Files:**

*   `cmd/`: Main application entry points (e.g., `cmd/server/main.go`).
*   `internal/`: Private application logic, including business logic, data access, and core functionalities.
*   `configs/`: Application configuration files (e.g., `feed_sources.json`).
*   `go.mod`, `go.sum`: Go module definitions and dependencies.
*   `web/`: Potentially contains frontend assets or templates served by the application (Needs confirmation by inspecting contents).
*   `.env`, `.env.example`: Environment variable configuration (Should not be committed directly, except for the example).

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
    *   `architecture.md`: Describes the data flow and architecture (e.g., CompositeScore calculation).
    *   `testing.md`: Comprehensive guide on running, analyzing, and debugging tests.
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
*   **Essential Tests (`scripts/test.cmd essential`)**: Cover core functionality described as essential for basic operation. *(Passes with workaround)*

**Important Note on Testing:** Due to concurrency issues with SQLite and background LLM analysis tasks triggered by some API calls, tests involving Newman (`backend`, `api`, `essential`, `all`, `debug`, `confidence`) currently require the `NO_AUTO_ANALYZE=true` environment variable to be set. The provided `test.cmd` and `test.sh` scripts handle this automatically. This prevents tests from failing due to database locks (`SQLITE_BUSY`) caused by contention between the test runner and background analysis.

Common commands using the consolidated test script:

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

### Running Locally

Start the server:
```
go run cmd/server/main.go
```

The server will be available at http://localhost:8080

## Contributing

1. Ensure tests pass locally
2. Add tests for new functionality
3. Update documentation as needed
4. Submit a pull request. See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

## License

MIT License - see LICENSE file for details