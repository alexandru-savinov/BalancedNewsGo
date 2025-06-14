# Phase 2 Implementation Guide - Quick Start

## 🎯 Phase 2 Overview
Phase 2 establishes robust, automated test infrastructure with proper orchestration and data management. This guide helps you get started with the newly implemented components.

## 📋 What Was Implemented

### 1. Comprehensive Test Runner (`scripts/comprehensive-test-runner.ps1`)
- **Purpose**: Orchestrates all test types with server lifecycle management
- **Features**: 
  - Automated server startup/shutdown
  - Support for unit, integration, E2E, and security tests
  - Coverage reporting
  - Test result aggregation
  - Proper cleanup and error handling

### 2. Database Testing Infrastructure (`internal/testing/database.go`)
- **Purpose**: Provides isolated database environments for testing
- **Features**:
  - SQLite in-memory databases for fast unit tests
  - PostgreSQL containers for integration tests
  - Automatic test data seeding and cleanup
  - Schema application and migration support

### 3. Test Environment Setup (`scripts/setup-test-env.ps1`)
- **Purpose**: Configures test environment variables and validates dependencies
- **Features**:
  - Environment variable configuration
  - Dependency validation
  - Test directory initialization
  - Configuration file validation

### 4. Test Configuration (`configs/test-config.json`)
- **Purpose**: Centralized configuration for test environments
- **Features**:
  - Database configuration options
  - Mock service settings
  - Test-specific parameters
  - Environment-specific overrides

## 🚀 Quick Start Guide

### Step 1: Install Dependencies
```powershell
# Install Go dependencies
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go mod tidy

# Verify installation
go build ./...
```

### Step 2: Set Up Test Environment
```powershell
# Set up test environment (SQLite-based, fast)
.\scripts\setup-test-env.ps1 -Verbose

# Or set up with containers (PostgreSQL-based, slower but more realistic)
.\scripts\setup-test-env.ps1 -UseContainers -Verbose
```

### Step 3: Validate Implementation
```powershell
# Run Phase 2 validation
.\scripts\validate-phase2.ps1 -Verbose

# Skip container tests if Docker is not available
.\scripts\validate-phase2.ps1 -SkipContainerTests
```

### Step 4: Run Tests
```powershell
# Run all tests
.\scripts\comprehensive-test-runner.ps1 -TestType all -Verbose -Coverage

# Run only unit tests (fast)
.\scripts\comprehensive-test-runner.ps1 -TestType unit -Verbose

# Run tests without starting server (if server is already running)
.\scripts\comprehensive-test-runner.ps1 -TestType unit -SkipServerStart
```

## 📝 Using the Database Testing Infrastructure

### Basic Usage Example
```go
package mypackage

import (
    "testing"
    "your-project/internal/testing"
)

func TestMyFunction(t *testing.T) {
    // Create test database (SQLite for speed)
    testDB := testing.NewTestDatabase(t, false)
    
    // Seed test data
    testDB.DB.SeedTestData(t)
    
    // Your test logic here
    // Database cleanup happens automatically
}
```

### Integration Test Example
```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Create PostgreSQL container for integration tests
    testDB := testing.NewTestDatabase(t, true)
    
    // Test complex database operations
    // Container cleanup happens automatically
}
```

## 🔧 Configuration Options

### Test Configuration (`configs/test-config.json`)
```json
{
  "database": {
    "type": "sqlite",           // or "postgres"
    "connection": ":memory:",   // or connection string
    "migrations_path": "./migrations",
    "seed_data": true
  },
  "testing": {
    "use_containers": false,    // Enable for integration tests
    "container_timeout": "60s",
    "cleanup_after_test": true,
    "parallel_tests": true
  }
}
```

### Environment Variables
```powershell
$env:NBG_ENV = "test"
$env:NBG_CONFIG_FILE = "configs/test-config.json"
$env:NBG_DATABASE_TYPE = "sqlite"  # or "postgres"
$env:NBG_LLM_MOCK = "true"
$env:NBG_RSS_MOCK = "true"
```

## 🧪 Test Types and Usage

### 1. Unit Tests
- **Purpose**: Fast, isolated tests using SQLite
- **Command**: `.\scripts\comprehensive-test-runner.ps1 -TestType unit`
- **Database**: In-memory SQLite
- **Speed**: Very fast (< 1 second per test)

### 2. Integration Tests
- **Purpose**: Tests with real database interactions
- **Command**: `.\scripts\comprehensive-test-runner.ps1 -TestType integration`
- **Database**: PostgreSQL container
- **Speed**: Slower (container startup overhead)

### 3. E2E Tests
- **Purpose**: Full application testing with UI
- **Command**: `.\scripts\comprehensive-test-runner.ps1 -TestType e2e`
- **Requirements**: Node.js, Playwright
- **Database**: Test server with configured database

### 4. Security Tests
- **Purpose**: Security vulnerability scanning
- **Command**: `.\scripts\comprehensive-test-runner.ps1 -TestType security`
- **Requirements**: gosec tool
- **Output**: JSON and human-readable reports

## 📊 Test Results and Coverage

### Test Results Location
- `test-results/` - All test outputs
- `test-results/unit-tests.json` - Unit test results
- `test-results/integration-tests.log` - Integration test logs
- `test-results/e2e-results/` - E2E test artifacts
- `test-results/server-logs/` - Server logs during tests

### Coverage Reports
- `coverage/unit-coverage.out` - Raw coverage data
- `coverage/coverage.html` - HTML coverage report
- `coverage/coverage-summary.txt` - Text coverage summary

## 🔍 Troubleshooting

### Common Issues

1. **Docker not available**
   - Use `-SkipContainerTests` flag
   - Stick to SQLite-based tests
   - Install Docker Desktop if needed

2. **Go module issues**
   - Run `go mod tidy`
   - Clear module cache: `go clean -modcache`
   - Verify Go version: `go version` (requires 1.21+)

3. **Test failures**
   - Check `test-results/test-summary.json`
   - Review individual test logs
   - Verify test environment setup

4. **Server startup issues**
   - Check if port 8080 is available
   - Verify server health endpoint
   - Review server logs in `test-results/server-logs/`

### Performance Tips

1. **Use SQLite for unit tests** - 10x faster than containers
2. **Use containers for integration tests** - More realistic
3. **Run tests in parallel** - Set `parallel_tests: true`
4. **Use coverage selectively** - Only when needed

## 🎯 Success Criteria

Phase 2 is successfully implemented when:

- [ ] ✅ All validation checks pass: `.\scripts\validate-phase2.ps1`
- [ ] ✅ Unit tests run successfully: `.\scripts\comprehensive-test-runner.ps1 -TestType unit`
- [ ] ✅ Test environment sets up correctly: `.\scripts\setup-test-env.ps1`
- [ ] ✅ Database tests work with both SQLite and PostgreSQL
- [ ] ✅ Coverage reports are generated properly
- [ ] ✅ Server lifecycle management works correctly

## ➡️ Next Steps

After Phase 2 completion:
1. **Proceed to Phase 3**: E2E Test Stabilization
2. **Integrate with CI/CD**: Add GitHub Actions workflow
3. **Expand test coverage**: Add more test cases
4. **Performance testing**: Add load testing capabilities

---

**Phase 2 Status**: ✅ IMPLEMENTED  
**Next Phase**: `04_phase3_e2e_stabilization.md`  
**Estimated Setup Time**: 15-30 minutes  
**Dependencies**: Go 1.21+, Docker (optional)
