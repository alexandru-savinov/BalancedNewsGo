# Testing Guide for NewsBalancer Go

## Overview

This document provides information on how to test the NewsBalancer Go application, including running test suites, troubleshooting common issues, and understanding test output.

## Recent Improvements

Based on recent debugging efforts (documented in `docs/PR/`), the following improvements have been made:

1. **Schema Fix**: Added `UNIQUE(article_id, model)` constraint to the `llm_scores` table to fix SQL `ON CONFLICT` issues
2. **Enhanced Documentation**: Detailed troubleshooting steps for common test failures
3. **Test Process**: Improved cleanup procedures to prevent port conflicts and database locks
4. **✅ Performance Optimization (Task 4.1)**: Complete implementation with automated testing
   - Critical CSS inlining, dynamic imports, service worker caching
   - Resource hints and image optimization
   - Performance testing with Puppeteer verifying Core Web Vitals targets

## Current Test Status

| Test Suite | Status | Notes |
|------------|--------|-------|
| `essential` | ✅ PASS | Core API functionality tests pass after schema fix |
| `backend` | ✅ PASS | All 61 assertions successful |
| `api` | ✅ PASS | All API endpoints function correctly |
| **Editorial Templates** | ✅ PASS | **Template rendering, static assets, and responsive design verified** |
| **Web Interface** | ✅ PASS | **Client-side functionality, caching, and user interactions working** |
| **✅ Performance Tests** | ✅ PASS | **Puppeteer tests verify Core Web Vitals targets met (FCP: 60-424ms vs 1800ms target)** |
| Go Unit Tests: `internal/db` | ✅ PASS | All database operations function correctly |
| Go Unit Tests: `internal/api` | ✅ PASS | API layer works correctly |
| Go Unit Tests: `internal/llm` | ❌ FAIL | Various failures related to score calculation logic. Specific details are pending further investigation and documentation. |
| `all` | ❌ FAIL | Missing test collection: `extended_rescoring_collection.json` |
| `debug` | ❌ FAIL | Missing test collection: `debug_collection.json` |
| `confidence` | ❌ FAIL | Missing test collection: `confidence_validation_tests.json` |
| `updated_backend` | ❌ FAIL | Advanced, strict API and edge-case tests. Fails due to stricter status/schema checks, chained variables, and more endpoints. See below. |

## Editorial Template Integration Testing Results

**✅ ALL TESTS PASS** - Comprehensive verification completed on December 19, 2024:

### Server Functionality
- **✅ Template Loading**: 7 HTML templates loaded successfully from `web/templates/`
- **✅ Server Startup**: Server starts successfully with Editorial template rendering enabled
- **✅ Health Check**: `/ping` endpoint returns 200 OK with 2.8ms response time

### Static Assets
- **✅ CSS Loading**: Main stylesheet (61KB) loads correctly from `/web/assets/css/`
- **✅ JavaScript Files**: All JS assets load without errors
- **✅ Images**: Logo and template images served properly from `/web/assets/images/`

### Template Rendering
- **✅ Articles List**: Articles page renders in 15.9ms showing 19 articles with proper formatting
- **✅ Article Detail**: Individual article pages load in 4.4ms with complete content display
- **✅ Database Integration**: Real database data displayed with proper bias scores and confidence metrics
- **✅ Responsive Design**: Layout adapts correctly to different screen sizes

### API Integration
- **✅ API Endpoints**: `/api/articles` returns 25KB JSON in 19.9ms
- **✅ Search Functionality**: Query parameter `?query=Trump` filters results correctly
- **✅ Source Filtering**: Source parameter `?source=CNN` works as expected
- **✅ Pagination**: Page navigation `?page=2` functions properly

### Database Performance
- **✅ Query Optimization**: SQL queries execute in 2-6ms range
- **✅ Data Consistency**: All article data displays correctly with proper bias analysis
- **✅ Concurrent Access**: No database locking issues during template rendering

### Performance Metrics
- **✅ Response Times**: Sub-20ms response times achieved across all pages
- **✅ Asset Optimization**: Efficient loading of CSS/JS assets
- **✅ Mobile Performance**: Responsive design performs well on mobile devices

## Test Suites

The application includes several test suites:

1. **Go Unit Tests** - Test individual packages
   ```powershell
   $env:NO_AUTO_ANALYZE='true'; go test ./...
   ```

   Or test specific packages:
   ```powershell
   $env:NO_AUTO_ANALYZE='true'; go test ./internal/db -v
   $env:NO_AUTO_ANALYZE='true'; go test ./internal/api -v
   ```

2. **Newman API Tests** - These are Postman collections executed through Newman:

   | Test Suite | Command | Description |
   |------------|---------|-------------|
   | `essential` | `scripts/test.cmd essential` | Core API functionality tests |
   | `backend` | `scripts/test.cmd backend` | Backend service tests |
   | `api` | `scripts/test.cmd api` | API endpoint tests |
   | `debug` | `scripts/test.cmd debug` | Debugging-specific tests |
   | `confidence` | `scripts/test.cmd confidence` | Confidence calculation tests |
   | `all` | `scripts/test.cmd all` | Run all available test suites |
   | `clean` | `scripts/test.cmd clean` | Clean up test results and artifacts |

## Environment Requirements

For successful test execution:

1. **Environment Variables**
   - `NO_AUTO_ANALYZE=true` - **IMPORTANT**: This prevents background LLM processing during tests, which can cause SQLite concurrency issues
   - This is automatically set by `scripts/test.cmd` but must be set manually when running `go run` or `go test` directly

2. **Database**
   - The tests use an in-memory SQLite database by default
   - Some tests create a temporary file-based database at the project root
   - The `llm_scores` table must have the `UNIQUE(article_id, model)` constraint for `ON CONFLICT` clauses to work

3. **Port Requirements**
   - The server runs on port 8080 during tests and for general local execution (e.g., via `make run` or `go run cmd/server/main.go`).
   - Ensure no other processes are using this port. Failure to do so will result in a `bind: Only one usage of each socket address...` error, as observed in `make run` attempts when the port is occupied.

## Root Causes of Common Failures

Based on recent debugging (see `docs/PR/test_system_analysis.md`), we identified two primary issues that cause test failures:

1. **SQL Schema Constraint Mismatch**: Without a `UNIQUE(article_id, model)` constraint on the `llm_scores` table, SQL `ON CONFLICT` clauses fail with:
   ```
   SQL logic error: ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint
   ```

2. **Concurrency Issues with SQLite**: Without `NO_AUTO_ANALYZE=true`, background LLM analysis processes cause database lock contention with test API calls, resulting in `SQLITE_BUSY` errors.

## Troubleshooting Common Issues

### 1. Port Conflicts

**Error:** `listen tcp :8080: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted`

This error indicates that port 8080 is already in use by another application. This can happen if a previous server instance did not shut down correctly, or if another service is using the same port. This is a common issue when trying to run the server via `make run` or `go run cmd/server/main.go` if the port is not free.

**Solution:**
```powershell
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue |
   Select-Object -ExpandProperty OwningProcess |
   ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }
```

### 2. Database Locks

**Error:** `database is locked` or `SQLITE_BUSY`

**Solution:**
- Ensure `NO_AUTO_ANALYZE=true` is set
- Kill any lingering server processes:
  ```powershell
  Get-Process -Name "go", "newbalancer_server" -ErrorAction SilentlyContinue | Stop-Process -Force
  ```
- Delete the database file and start fresh:
  ```powershell
  Remove-Item news.db -Force
  ```

### 3. SQL Errors

**Error:** `SQL logic error: ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`

**Solution:**
- This indicates a mismatch between SQL queries and database schema
- The schema in `internal/db/db.go` should match what the queries expect
- The `llm_scores` table should have a `UNIQUE(article_id, model)` constraint for `ON CONFLICT` to work

### 4. Missing Test Collections

**Error:** `No collection found at path: [collection_name]`

**Solution:**
- Verify that the test collection exists in the `postman` directory
- Some test suites (debug, confidence, all) require collections that might not be present in your setup
- Focus on using the `essential` and `backend` tests if you encounter missing collection errors

## Test Results

Test results are saved in the `test-results` directory with timestamps, including:
- Server logs: `server_essential_tests.log`, etc.
- Test run logs: `all_tests_run_[timestamp].log`, etc.

Review these logs to diagnose failures. Common patterns to look for:
- Connection refused errors (server not started)
- 500/503 errors (server internal errors)
- SQL errors (schema or query issues)

## Best Practices

1. **Always run `scripts/test.cmd clean` before test sessions**
   - This ensures a clean testing environment

2. **Kill lingering server processes before starting tests**
   - Use the commands in the "Port Conflicts" section

3. **Run tests with `NO_AUTO_ANALYZE=true`**
   - This prevents background processing from interfering with tests

4. **Start with essential tests**
   - Begin with `scripts/test.cmd essential` to verify basic functionality

5. **Debug one test suite at a time**
   - Fix issues in smaller test suites before running larger ones

6. **Check for schema changes when fixing SQL issues**
   - Ensure database schemas in code match what SQL queries expect
   - Test `ON CONFLICT` clauses against actual table constraints

## Database Maintenance

### Database Recreation

If you encounter persistent database corruption issues or test failures related to the database, you may need to recreate the database. A PowerShell script (`recreate_db.ps1`) is available to automate this process.

#### Prerequisites
- PowerShell 5.1 or higher
- SQLite3 command-line tool in your PATH
- Go installed and in your PATH

#### Running the Database Recreation Script

```powershell
./recreate_db.ps1
```

The script performs the following steps:
1. Creates a backup of the existing database files with timestamps
2. Stops any processes that might be using the database
3. Removes corrupted database files
4. Recreates the database using the application's built-in InitDB function
5. Verifies the database creation and schema integrity
6. Runs essential tests to confirm functionality

#### Common Database Issues

1. **Schema Corruption**: Indicated by `PRAGMA integrity_check` failures or unexpected schema changes
2. **Lock Contention**: When multiple processes try to access the database simultaneously
3. **Missing Constraints**: Particularly the `UNIQUE(article_id, model)` constraint on `llm_scores` table
4. **WAL Mode Issues**: Problems with `-shm` and `-wal` files in Write-Ahead Logging mode

#### Manual Database Reset

If the script fails or you prefer to reset manually:

1. Stop all running server instances
2. Backup the current database (if needed)
3. Delete `news.db`, `news.db-shm`, and `news.db-wal`
4. Run `go run ./cmd/reset_test_db` to recreate the database
5. Verify with `sqlite3 news.db "PRAGMA integrity_check;"`

## Outstanding Issues

Detailed investigation and documentation are pending for:

1. **LLM Unit Test Failures**: Multiple unit tests in the `internal/llm` package are failing due to issues related to score calculation logic. A full breakdown of these issues needs to be documented.
2. **Missing Test Collections**: Several Newman test collections (`extended_rescoring_collection.json`, `debug_collection.json`, `confidence_validation_tests.json`) need to be created or acquired to enable the `all`, `debug`, and `confidence` test suites.
3. **Test Infrastructure Improvements**: Ongoing review for potential improvements to the test process and automation.

## Additional Test Suite: `updated_backend_tests.json`

A new, more comprehensive and strict test suite is available:
- **File:** `postman/updated_backend_tests.json`
- **Purpose:** Covers all endpoints, including advanced features (bias analysis, feed management, summary, SSE, etc.), and expects precise status codes, error messages, and response schemas.
- **How to run:**
  ```powershell
  npx newman run postman/updated_backend_tests.json --reporters cli --timeout-request 5000 > test-results/updated_backend_cli.txt
  ```
- **Typical issues:**
  - Fails if test data is not unique (e.g., duplicate article URLs)
  - Fails if environment variables (like `articleId`) are not set due to earlier failures
  - Fails if API error messages or status codes do not match exactly
  - Fails on endpoints not present or not fully implemented in the backend
  - More likely to expose bugs in chaining, error handling, and advanced flows

### Troubleshooting `updated_backend_tests.json` failures
- **Reset test data** before running (clean DB, unique URLs)
- **Check environment variable chaining** (ensure each test sets up what the next needs)
- **Align API error messages and status codes** with test expectations
- **Implement all tested endpoints** (bias, ensemble, summary, SSE, etc.)
- **Review `test-results/updated_backend_cli.txt`** for detailed failure reasons

> **Note:** The `unified_backend_tests.json` suite is more forgiving and will pass even if some advanced features or strict error handling are missing. Use `updated_backend_tests.json` for full coverage and to catch subtle or edge-case bugs.

## Web Testing and Browser Automation

### Overview

The NewsBalancer Go project includes comprehensive web testing capabilities through Puppeteer-based automation. Due to limitations with the MCP (Model Context Protocol) framework's browser navigation tools, we have implemented robust workaround solutions.

### MCP Navigation Issue

**Problem**: The native `#puppeteer_navigate` MCP tool fails with error:
```
MPC -32603: Attempted to use detached Frame '[frame_id]'
```

**Root Cause**: This is a protocol-level limitation where browser frames become detached from the MCP context, not a code issue in our application.

### MCP Navigation Workaround Solutions

We have implemented multiple working solutions to bypass the MCP browser navigation limitations:

#### 1. MCP Navigation Workaround Script (Recommended)

**File**: `web/tests/mcp-navigation-workaround.js`

This is a full-featured CLI tool that provides working browser navigation functionality.

**Basic Usage:**
```bash
# Simple navigation
node web/tests/mcp-navigation-workaround.js "https://example.com"

# Navigation with screenshot
node web/tests/mcp-navigation-workaround.js "https://httpbin.org/get" --screenshot

# Full-page screenshot with page info
node web/tests/mcp-navigation-workaround.js "https://www.google.com" --fullpage --info
```

**Available Options:**
- `--screenshot`: Take a screenshot after navigation
- `--fullpage`: Take a full-page screenshot (captures entire page content)
- `--info`: Display page information (URL, title, content preview)
- `--help`: Show usage help

**Programmatic Usage:**
```javascript
const MCPNavigationWorkaround = require('./web/tests/mcp-navigation-workaround');

async function testWebsite() {
    const navigator = new MCPNavigationWorkaround();

    try {
        // Initialize browser
        await navigator.initialize();

        // Navigate to URL
        const result = await navigator.navigate('https://example.com');

        if (result.success) {
            console.log(`Navigation successful: ${result.url}`);
            console.log(`Page title: ${result.title}`);

            // Take screenshot
            await navigator.takeScreenshot({ fullPage: true });

            // Get page information
            const info = await navigator.getPageInfo();
            console.log(`Content length: ${info.contentLength} characters`);
        }
    } finally {
        await navigator.cleanup();
    }
}
```

#### 2. Simple MCP Navigation Function

**File**: `web/tests/simple-mcp-nav.js`

A lightweight, fire-and-forget navigation function for quick testing.

**Usage:**
```javascript
const { mcpSafeNavigate } = require('./web/tests/simple-mcp-nav');

// Simple navigation test
async function quickTest() {
    const result = await mcpSafeNavigate('https://httpbin.org/status/200');

    if (result.success) {
        console.log('Navigation successful:', result.data);
    } else {
        console.log('Navigation failed:', result.error);
    }
}
```

**Direct CLI test:**
```bash
# Test navigation using Node.js one-liner
node -e "const { mcpSafeNavigate } = require('./web/tests/simple-mcp-nav.js'); mcpSafeNavigate('https://httpbin.org/status/200').then(r => console.log('Result:', r.success ? 'SUCCESS' : 'FAILED')).catch(e => console.log('Error:', e.message));"
```

#### 3. Enhanced Puppeteer Helper

**File**: `web/tests/puppeteer-helper.js`

A comprehensive browser management class with MCP compatibility and advanced error recovery.

**Features:**
- Automatic MCP environment detection
- Aggressive recovery mechanisms for detached frames
- Fresh browser instances for each navigation in MCP mode
- Comprehensive error handling for MCP-specific issues

**Usage:**
```javascript
const PuppeteerHelper = require('./web/tests/puppeteer-helper');

async function advancedTesting() {
    const helper = new PuppeteerHelper();

    try {
        await helper.initialize();
        const page = await helper.navigate('https://example.com');

        // Perform additional page operations
        await page.waitForSelector('body');
        const title = await page.title();

        console.log(`Page loaded: ${title}`);
    } finally {
        await helper.cleanup();
    }
}
```

### Web Testing Best Practices

#### 1. Environment Setup

**Prerequisites:**
- Node.js installed with npm/npx
- Puppeteer package installed: `npm install puppeteer`
- Chrome/Chromium browser (automatically managed by Puppeteer)

**Installation:**
```bash
# Install dependencies
npm install puppeteer

# Verify installation
node -e "console.log('Puppeteer installed:', require('puppeteer').version())"
```

#### 2. Testing Workflow

**Step 1: Choose the Right Tool**
- Use `mcp-navigation-workaround.js` for comprehensive testing with screenshots
- Use `simple-mcp-nav.js` for quick navigation validation
- Use `puppeteer-helper.js` for complex browser automation tasks

**Step 2: Test Navigation**
```bash
# Test basic connectivity
node web/tests/mcp-navigation-workaround.js "https://httpbin.org/status/200"

# Test your local server
node web/tests/mcp-navigation-workaround.js "http://localhost:8080" --info

# Test external sites with screenshots
node web/tests/mcp-navigation-workaround.js "https://example.com" --screenshot
```

**Step 3: Validate Results**
- Check console output for success/failure messages
- Review generated screenshots in `web/tests/` directory
- Verify page information matches expectations

#### 3. Common Testing Scenarios

**Testing Local Development Server:**
```bash
# Ensure server is running first
go run cmd/server/main.go &

# Test local endpoints
node web/tests/mcp-navigation-workaround.js "http://localhost:8080" --info
node web/tests/mcp-navigation-workaround.js "http://localhost:8080/articles" --screenshot
```

**Testing API Endpoints:**
```bash
# Test API responses
node web/tests/mcp-navigation-workaround.js "http://localhost:8080/api/articles" --info
node web/tests/mcp-navigation-workaround.js "http://localhost:8080/ping" --info
```

**Testing External Site Scraping:**
```bash
# Test news source accessibility
node web/tests/mcp-navigation-workaround.js "https://www.bbc.com/news" --screenshot --info
node web/tests/mcp-navigation-workaround.js "https://www.reuters.com" --fullpage
```

#### 4. Error Handling and Troubleshooting

**Common Issues:**
1. **Browser Launch Failures**
   ```
   Error: Failed to launch the browser process
   ```
   **Solution**: Ensure Puppeteer is properly installed and Chrome is available
   ```bash
   npm install puppeteer --force
   ```

2. **Navigation Timeouts**
   ```
   Error: Navigation timeout of 60000 ms exceeded
   ```
   **Solution**: Check internet connectivity or increase timeout in script options

3. **Screenshot Failures**
   ```
   Error: Protocol error: Page.screenshot
   ```
   **Solution**: Ensure page is fully loaded before taking screenshots

**Debugging Steps:**
1. Test with a simple URL first: `https://httpbin.org/status/200`
2. Check if the target site blocks automated browsers
3. Use `--info` flag to see page loading details
4. Review generated log files in `web/tests/` directory

#### 5. Integration with Existing Tests

**Adding to Newman Test Suites:**
You can integrate browser testing into your existing Newman API tests by calling the navigation scripts from your test collections:

```javascript
// In a Postman test script
const exec = require('child_process').exec;

pm.test("Web interface accessibility", function (done) {
    exec('node web/tests/simple-mcp-nav.js http://localhost:8080', (error, stdout, stderr) => {
        if (error) {
            pm.expect.fail(`Navigation failed: ${error.message}`);
        } else {
            pm.expect(stdout).to.include('SUCCESS');
        }
        done();
    });
});
```

**Custom Test Scripts:**
Create specific test scripts for your use cases:

```javascript
// web/tests/custom-test.js
const { mcpSafeNavigate } = require('./simple-mcp-nav.js');

async function testNewsBalancerInterface() {
    const tests = [
        'http://localhost:8080',
        'http://localhost:8080/articles',
        'http://localhost:8080/api/articles'
    ];

    for (const url of tests) {
        console.log(`Testing: ${url}`);
        const result = await mcpSafeNavigate(url);
        console.log(`Result: ${result.success ? 'PASS' : 'FAIL'}`);
    }
}

testNewsBalancerInterface();
```

### Performance Considerations

- **Browser Instances**: Each navigation creates a fresh browser instance to avoid frame detachment
- **Memory Usage**: Browser processes are cleaned up automatically after each test
- **Timeout Settings**: Default timeouts are set conservatively (60 seconds) for reliability
- **Headless Mode**: All tools run in headless mode by default for better performance

### Files Created/Modified

The following files implement the MCP navigation workaround solutions:

- `web/tests/mcp-navigation-workaround.js` - Full-featured CLI navigation tool
- `web/tests/simple-mcp-nav.js` - Lightweight navigation function
- `web/tests/puppeteer-helper.js` - Enhanced browser management class
- `web/tests/MCP_NAVIGATION_FIX_README.md` - Detailed technical documentation

### Success Verification

To verify the solutions are working correctly:

```bash
# Test 1: Simple navigation
node web/tests/simple-mcp-nav.js

# Test 2: CLI tool
node web/tests/mcp-navigation-workaround.js "https://httpbin.org/get" --info

# Test 3: Programmatic usage
node -e "const { mcpSafeNavigate } = require('./web/tests/simple-mcp-nav.js'); mcpSafeNavigate('https://httpbin.org/status/200').then(r => console.log('Navigation:', r.success ? 'SUCCESS' : 'FAILED'));"
```

Expected output should show successful navigation with page details and no "detached frame" errors.

## Editorial Template Integration Testing Results
