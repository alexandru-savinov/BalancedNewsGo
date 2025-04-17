# CLI Testing Guide

This guide explains how to run and analyze tests using the command line interface (CLI) without needing to open Postman.

## Prerequisites

- Node.js installed
- Go installed
- Newman installed (`npm install -g newman`)

## Running Tests

All testing operations can be performed using the `test` command:

```
test <command>
```

or using npm:

```
npm run test:<command>
```

### Available Commands

- `test backend` - Run backend fixes tests
- `test all` - Run all tests
- `test debug` - Run debug tests
- `test report` - Generate HTML test report
- `test analyze` - Analyze test results
- `test list` - List test result files
- `test clean` - Clean test results
- `test help` - Show help message

## Examples

### Running Backend Tests

```
test backend
```

or

```
npm run test:backend
```

### Running All Tests

```
test all
```

or

```
npm run test:all
```

### Analyzing Test Results

```
test analyze
```

or

```
npm run test:analyze
```

This will show a list of available test result files and allow you to select one for detailed analysis.

### Generating HTML Report

```
test report
```

or

```
npm run test:report
```

This will generate an HTML report of all test results in the `test-results` directory.

## Running Essential Rescoring Tests (Critical Path)

To run the most important backend rescoring tests (including all boundary and edge cases), use the following script:

```
run_essential_tests.cmd
```

This will:
- Start the backend server (if not already running)
- Run the essential Postman collection for rescoring
- Save results to `test-results/essential_rescoring_tests.json`
- Exit with the test result code (0 = all pass)

## Troubleshooting
- Ensure you use unique article titles/URLs in your tests to avoid data collisions.
- Test results are saved in the `test-results/` directory for review and analysis.

## Test Result Analysis

The test analyzer provides a powerful way to examine test results without opening Postman:

1. **List Test Files**:
   ```
   test list
   ```
   Shows all available test result files.

2. **Analyze All Test Results**:
   ```
   test analyze
   ```
   Provides a summary of all test results and allows you to select a specific file for detailed analysis.

3. **Direct Analysis**:
   ```
   node analyze_test_results.js analyze <filename>
   ```
   Analyzes a specific test result file.

## Understanding Test Output

The test output includes:

1. **Summary Information**:
   - Total tests run
   - Number of passed/failed tests
   - Execution time

2. **Detailed Results** (when requested):
   - Request details (method, URL)
   - Response status and time
   - Assertions (passed/failed)
   - Response body
   - Error messages for failed tests

## Debugging Failed Tests

When tests fail, the analyzer provides detailed information to help you debug the issues:

1. Run the debug tests:
   ```
   test debug
   ```

2. Analyze the results:
   ```
   test analyze
   ```

3. Select the debug test results file for detailed analysis.

4. Look for:
   - Failed assertions
   - Error messages
   - Unexpected response bodies
   - Status codes

## Common Issues and Solutions

1. **Feedback Table Schema Issues**:
   - The feedback table should have the following columns: id, article_id, user_id, feedback_text, category, ensemble_output_id, source, created_at.
   - If any of these columns are missing, the feedback submission will fail.

2. **Response Format Issues**:
   - The API returns responses in a standard format: `{ "success": true/false, "data": {...} }` or `{ "success": false, "error_message": "..." }`.
   - Make sure your tests are checking the correct fields in the response.

3. **Cache Issues**:
   - The getArticlesHandler uses caching to improve performance.
   - If you're not seeing updated data, try adding a unique query parameter to bypass the cache.

4. **Ensemble Details Issues**:
   - The ensemble details endpoint requires that the article has been processed by the LLM.
   - If you're getting a 404, it might be because the article hasn't been processed yet.