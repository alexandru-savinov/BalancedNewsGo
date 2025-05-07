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

Tests are primarily run using scripts located in the project root or via `npm` / `Makefile` commands.

### Using Root Scripts (`.cmd` for Windows, `.sh` for Linux/macOS)

These scripts execute specific test suites:

- **Unit Tests (Go):**
  ```bash
  go test ./...
  ```

- **Backend Integration / Fixes Tests:** Verifies core backend logic and recent fixes.
  ```bash
  # Windows
  ./run_backend_tests.cmd

  # Linux/macOS
  ./run_backend_tests.sh # (or equivalent npm run test:backend)
  ```

- **All Tests (including E2E):** Runs the most comprehensive suite, including backend, API, and potentially other tests like rescoring.
  ```bash
  # Windows
  ./run_all_tests.cmd

  # Linux/macOS
  ./run_all_tests.sh # (or equivalent npm run test:all)
  ```

- **Debug Tests:** Runs tests with more verbose output, saving detailed results for debugging.
  ```bash
  # Windows
  ./run_debug_tests.cmd

  # Linux/macOS
  # (Equivalent npm run test:debug might exist)
  ```

- **Essential Rescoring Tests:** Focuses on critical path tests for the rescoring logic.
  ```bash
  # Windows
  ./run_essential_tests.cmd

  # Linux/macOS
  ./run_essential_tests.sh
  ```
  This script typically starts the server, runs the tests, saves results to `test-results/essential_rescoring_tests.json`, and exits with a status code.

### Using NPM / Test Runner Commands

If configured in `package.json`, you might use `npm` or a custom `test` command:

```bash
# Examples (check package.json for actual commands)
npm run test:backend
npm run test:all
npm run test:debug
npm run test:report # Generate HTML report
npm run test:analyze # Analyze results via CLI
```

Alternatively, a `test` command might exist:

```bash
# Examples (check implementation for actual commands)
test backend
test all
test debug
test report
test analyze
test list
test clean
test help
```

## Analyzing and Debugging Results

### Generating Reports

An HTML report summarizing test runs can often be generated:

```bash
npm run report # (or test report)
```
This usually saves a report to the `test-results/` directory (e.g., `test_report.html`).

### Command-Line Analysis

You can analyze Newman/Postman results directly from the CLI without opening Postman, typically using a helper script:

1.  **List Result Files:** See available JSON results in `test-results/`.
    ```bash
    test list # (or similar command)
    ```
2.  **Analyze Results:** Run the analyzer, which might prompt you to choose a file or analyze the latest.
    ```bash
    test analyze # (or npm run test:analyze)
    ```
3.  **Direct Analysis:** Analyze a specific file.
    ```bash
    node analyze_test_results.js analyze <filename.json> # (Adjust script name/path as needed)
    ```

The analysis output usually includes:
- Summary (Total, Passed, Failed, Time)
- Detailed results for failures (Request, Response, Assertions, Errors)

### Debugging in Postman

For in-depth debugging, you can import the raw Newman result files (`.json` files saved in `test-results/`) into Postman:

1.  Run the desired test suite (especially `run_debug_tests.cmd` which saves `debug_tests.json`).
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

1.  **Feedback Table Schema Issues**:
    *   Ensure the `feedback` table has all required columns (e.g., `id`, `article_id`, `user_id`, `feedback_text`, `category`, `ensemble_output_id`, `source`, `created_at`). Missing columns cause submission failures.
    *   Use the `/api/debug/schema` endpoint (if available) or DB tools to verify the schema.
2.  **Response Format Issues**:
    *   The API likely follows a standard response structure (e.g., `{ "success": true/false, "data": {...} }` or `{ "success": false, "error_message": "..." }`).
    *   Verify tests are asserting against the correct fields and structure based on the actual API response (visible in Postman or CLI analysis).
3.  **Cache Issues**:
    *   Endpoints like `GET /articles` might use caching.
    *   If tests seem to get stale data, try adding a unique query parameter (e.g., `?timestamp=<value>`) to the request URL to bypass the cache during debugging.
4.  **Ensemble Details Issues**:
    *   Endpoints retrieving ensemble or detailed scores (e.g., `/api/articles/{id}/details`) might require the article to have been fully processed and scored by the LLMs.
    *   A `404 Not Found` might indicate the scoring process hasn't completed for that article yet.
5.  **Data Collisions:**
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