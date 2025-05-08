# Testing Guide

This guide explains how to set up your environment, run the various test suites, and analyze/debug test results for the NewsBalancer project.

## Prerequisites

Before running tests, ensure you have the following installed:

- Go (Version 1.24+ recommended, check `README.md` for the specific version)
- Node.js (Required for test runners and reporting tools)
- Newman (The Postman CLI runner): Install globally via npm:
  ```bash
  npm install -g newman
  ```

## Running Tests

Tests are primarily run using the consolidated test script (`scripts/test.cmd` for Windows, `scripts/test.sh` for Linux/macOS) or standard Go commands.

### Using the Consolidated Test Script

This script provides various subcommands to run different test suites. Use `scripts/test.cmd help` or `scripts/test.sh help` to see all available commands.

Common commands:

- **Go Unit Tests:**
  ```bash
  go test ./...
  ```
  *(Note: This command is run directly, not via the test script)*

- **Backend Integration Tests:**
  ```bash
  # Windows
  scripts/test.cmd backend
  # Linux/macOS
  scripts/test.sh backend 
  ```

- **API Tests:**
  ```bash
  # Windows
  scripts/test.cmd api
  # Linux/macOS
  scripts/test.sh api 
  ```

- **Essential Tests:**
  ```bash
  # Windows
  scripts/test.cmd essential
  # Linux/macOS
  scripts/test.sh essential 
  ```

- **All Tests (Combined):** Runs essential, extended, and confidence tests.
  ```bash
  # Windows
  scripts/test.cmd all
  # Linux/macOS
  scripts/test.sh all 
  ```

- **Debug Tests:** Runs a specific debug collection.
  ```bash
  # Windows
  scripts/test.cmd debug
  # Linux/macOS
  scripts/test.sh debug 
  ```

- **Confidence Validation Tests:**
  ```bash
  # Windows
  scripts/test.cmd confidence
  # Linux/macOS
  scripts/test.sh confidence 
  ```

- **Generate Report:** Creates an HTML report from latest results.
  ```bash
  # Windows
  scripts/test.cmd report
  # Linux/macOS
  scripts/test.sh report 
  ```

- **Analyze Results:** Runs a CLI analysis tool.
  ```bash
  # Windows
  scripts/test.cmd analyze
  # Linux/macOS
  scripts/test.sh analyze 
  ```

- **List Results:** Shows existing result files.
  ```bash
  # Windows
  scripts/test.cmd list
  # Linux/macOS
  scripts/test.sh list 
  ```

- **Clean Results:** Deletes generated test results and logs.
  ```bash
  # Windows
  scripts/test.cmd clean
  # Linux/macOS
  scripts/test.sh clean 
  ```

### Important Note on Concurrency

Due to concurrency limitations with SQLite and background LLM analysis tasks triggered by some API calls, tests involving Newman (`backend`, `api`, `essential`, `all`, `debug`, `confidence`) currently require the `NO_AUTO_ANALYZE=true` environment variable to be set. The provided `test.cmd` and `test.sh` scripts handle this automatically by setting the variable before starting the Go server for the test run. This prevents tests from failing due to database locks (`SQLITE_BUSY`) caused by contention between the test runner and background analysis.

## Analyzing and Debugging Results

### Generating Reports

An HTML report summarizing test runs can be generated using the test script:

```bash
# Windows
scripts/test.cmd report
# Linux/macOS
scripts/test.sh report
```
This usually saves a report to the `test-results/` directory (e.g., `test_report.html`).

### Command-Line Analysis

You can analyze Newman/Postman results directly from the CLI using the test script:

1.  **List Result Files:** See available JSON results in `test-results/`.
    ```bash
    scripts/test.cmd list
    ```
2.  **Analyze Results:** Run the analyzer, which might prompt you to choose a file or analyze the latest.
    ```bash
    scripts/test.cmd analyze
    ```

The analysis output usually includes:
- Summary (Total, Passed, Failed, Time)
- Detailed results for failures (Request, Response, Assertions, Errors)

### Debugging in Postman

For in-depth debugging, you can import the raw Newman result files (`.json` files saved in `test-results/`) into Postman:

1.  Run the desired test suite (especially `scripts/test.cmd debug` which saves `debug_tests.json`).
2.  Open Postman.
3.  Click the "Import" button.
4.  Select the "File" tab and upload the relevant `.json` file from `test-results/`.
5.  The results import as a temporary collection.
6.  Examine individual requests:
    *   View request details (headers, body).
    *   View the actual response received (body, headers, status code).
    *   Check the "Test Results" tab for assertion passes/failures.
    *   Look at the Postman Console (`Ctrl+Alt+C` or `Cmd+Option+C`) for detailed logs written by test scripts (`pm.test`, `console.log`).
7.  You can modify and re-run individual requests directly within Postman to pinpoint issues.

### Debug Endpoints

The API may include specific endpoints to aid debugging:

- `GET /api/debug/schema`: Check if this exists; it might return the current database schema and sample data.

## Common Issues and Solutions

When tests fail, consider these common areas:

1.  **Database Locking (`SQLITE_BUSY`):**
    *   If you encounter errors related to the database being locked, ensure you are running tests via the `scripts/test.cmd` or `scripts/test.sh` scripts. These scripts set the `NO_AUTO_ANALYZE=true` environment variable, which prevents background LLM analysis from causing database contention during tests.
    *   If running Newman manually or via other means, you may need to set this environment variable yourself.

2.  **Feedback Table Schema Issues**:
    *   Ensure the `feedback` table has all required columns (e.g., `id`, `article_id`, `user_id`, `feedback_text`, `category`, `ensemble_output_id`, `source`, `created_at`). Missing columns cause submission failures.
    *   Use the `/api/debug/schema` endpoint (if available) or DB tools to verify the schema.
3.  **Response Format Issues**:
    *   The API likely follows a standard response structure (e.g., `{ "success": true/false, "data": {...} }` or `{ "success": false, "error_message": "..." }`).
    *   Verify tests are asserting against the correct fields and structure based on the actual API response (visible in Postman or CLI analysis).
4.  **Cache Issues**:
    *   Endpoints like `GET /articles` might use caching.
    *   If tests seem to get stale data, try adding a unique query parameter (e.g., `?timestamp=<value>`) to the request URL to bypass the cache during debugging.
5.  **Ensemble Details Issues**:
    *   Endpoints retrieving ensemble or detailed scores (e.g., `/api/articles/{id}/details`) might require the article to have been fully processed and scored by the LLMs.
    *   A `404 Not Found` might indicate the scoring process hasn't completed for that article yet.
6.  **Data Collisions:**
    *   Ensure tests creating data (e.g., articles, feedback) use unique identifiers (titles, URLs, user IDs) per test run if the database isn't cleaned between runs, to avoid conflicts.

## Updating Tests

If API behavior changes, the Postman tests need updating:

1.  Open the source Postman Collection (likely stored in the `postman/` directory).
2.  Modify the requests or test scripts (`Tests` tab in Postman) as needed.
3.  Save the changes in Postman.
4.  Export the *entire collection* from Postman (usually as `Collection v2.1`).
5.  Replace the corresponding collection file in the `postman/` directory with the newly exported file.
6.  Commit the updated collection file to version control.
7.  Run the tests again using Newman (via scripts or CLI) to ensure they pass with the updated collection. 