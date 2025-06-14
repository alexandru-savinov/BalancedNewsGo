# Phase 2: Test Infrastructure Implementation Guide
*Priority*: P0 - CRITICAL | *Estimated Time*: 1-2 hours | *Dependencies*: Phase 1 Complete

## 📋 Phase Overview

**Objective**: Establish robust, automated test infrastructure with proper orchestration and data management  
**Success Criteria**: Automated test execution, reliable test environment, clean test data management  
**Next Phase**: Proceed to `04_phase3_e2e_stabilization.md`

### What This Phase Accomplishes
- **✅ Test Orchestration**: Comprehensive test runner with server management
- **✅ Database Testing**: Proper test database setup and cleanup using Testcontainers
- **✅ Environment Management**: Consistent test environment configuration
- **✅ CI/CD Ready**: Foundation for automated pipeline integration

## 🔧 Implementation Steps

### Step 2.1: Automated Test Orchestration *(45 minutes)*

#### Enhanced Test Runner with Server Lifecycle Management
Building on Phase 1's basic server management, create a comprehensive test orchestration system:

**File**: `scripts/comprehensive-test-runner.ps1`

```powershell
# Comprehensive Test Orchestration Script
param(
    [ValidateSet("all", "unit", "integration", "e2e", "security", "load")]
    [string]$TestType = "all",
    [switch]$SkipServerStart,
    [switch]$Verbose,
    [switch]$Coverage,
    [int]$TimeoutMinutes = 10
)

$ErrorActionPreference = "Stop"
$ResultsDir = "test-results"
$CoverageDir = "coverage"

# Logging with timestamps and colors
function Write-TestLog {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch($Level) {
        "ERROR" { "Red" }
        "WARN" { "Yellow" } 
        "SUCCESS" { "Green" }
        "DEBUG" { "Cyan" }
        default { "White" }
    }
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

# Enhanced server health check with retry logic
function Test-ServerHealth {
    param([int]$MaxRetries = 5, [int]$DelaySeconds = 2)
    
    for ($i = 1; $i -le $MaxRetries; $i++) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/healthz" -TimeoutSec 5 -Method Get
            if ($response.status -eq "ok" -or $response -eq "OK") {
                Write-TestLog "Server health check passed (attempt $i/$MaxRetries)" "SUCCESS"
                return $true
            }
        }
        catch {
            Write-TestLog "Health check attempt $i/$MaxRetries failed: $($_.Exception.Message)" "DEBUG"
        }
        
        if ($i -lt $MaxRetries) {
            Start-Sleep -Seconds $DelaySeconds
        }
    }
    
    Write-TestLog "Server health check failed after $MaxRetries attempts" "ERROR"
    return $false
}

# Advanced server startup with process management
function Start-TestServer {
    if ($SkipServerStart) {
        Write-TestLog "Skipping server startup as requested" "INFO"
        return $null
    }
    
    Write-TestLog "Checking existing server..." "INFO"
    if (Test-ServerHealth -MaxRetries 1) {
        Write-TestLog "Server already running and healthy" "SUCCESS"
        return $null
    }
    
    # Kill any existing server processes
    Get-Process | Where-Object { $_.ProcessName -eq "go" -and $_.CommandLine -like "*cmd/server*" } | ForEach-Object {
        Write-TestLog "Terminating existing server process (PID: $($_.Id))" "WARN"
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
    
    Write-TestLog "Starting new server process..." "INFO"
    
    # Create server logs directory
    $serverLogsDir = "$ResultsDir/server-logs"
    New-Item -ItemType Directory -Path $serverLogsDir -Force | Out-Null
    
    try {
        # Start server with output redirection
        $processStartInfo = New-Object Process
        $processStartInfo.StartInfo.FileName = "go"
        $processStartInfo.StartInfo.Arguments = "run ./cmd/server"
        $processStartInfo.StartInfo.UseShellExecute = $false
        $processStartInfo.StartInfo.RedirectStandardOutput = $true
        $processStartInfo.StartInfo.RedirectStandardError = $true
        $processStartInfo.StartInfo.CreateNoWindow = $true
        
        $serverProcess = [System.Diagnostics.Process]::Start($processStartInfo)
        
        # Wait for server to be ready
        $timeout = (Get-Date).AddMinutes($TimeoutMinutes)
        $serverReady = $false
        
        while ((Get-Date) -lt $timeout -and !$serverReady) {
            Start-Sleep -Seconds 2
            $serverReady = Test-ServerHealth -MaxRetries 1
            
            if (!$serverProcess.HasExited -eq $false) {
                throw "Server process exited unexpectedly"
            }
        }
        
        if ($serverReady) {
            Write-TestLog "Server started successfully (PID: $($serverProcess.Id))" "SUCCESS"
            return $serverProcess
        } else {
            throw "Server failed to start within $TimeoutMinutes minutes"
        }
    }
    catch {
        Write-TestLog "Server startup failed: $($_.Exception.Message)" "ERROR"
        throw
    }
}

# Test execution functions
function Invoke-UnitTests {
    Write-TestLog "Running Go unit tests..." "INFO"
    
    $testArgs = @("test", "./...")
    
    if ($Verbose) {
        $testArgs += "-v"
    }
    
    if ($Coverage) {
        New-Item -ItemType Directory -Path $CoverageDir -Force | Out-Null
        $testArgs += "-coverprofile=$CoverageDir/unit-coverage.out"
        $testArgs += "-covermode=atomic"
    }
    
    $testArgs += "-json"
    
    & go @testArgs | Tee-Object -FilePath "$ResultsDir/unit-tests.json"
    return $LASTEXITCODE
}

function Invoke-IntegrationTests {
    Write-TestLog "Running integration tests..." "INFO"
    
    # Example integration test command - adjust based on your setup
    $testCommand = @("test", "-tags=integration", "./tests/integration/...")
    
    if ($Verbose) {
        $testCommand += "-v"
    }
    
    & go @testCommand | Tee-Object -FilePath "$ResultsDir/integration-tests.log"
    return $LASTEXITCODE
}

function Invoke-E2ETests {
    Write-TestLog "Running E2E tests..." "INFO"
    
    # Ensure Playwright dependencies are available
    if (Get-Command "npx" -ErrorAction SilentlyContinue) {
        npx playwright test --reporter=json --output-dir="$ResultsDir/e2e-results" | Tee-Object -FilePath "$ResultsDir/e2e-tests.log"
        return $LASTEXITCODE
    } else {
        Write-TestLog "npx not found - skipping E2E tests" "WARN"
        return 0
    }
}

function Invoke-SecurityTests {
    Write-TestLog "Running security tests..." "INFO"
    
    if (Get-Command "gosec" -ErrorAction SilentlyContinue) {
        gosec -fmt json -out "$ResultsDir/security-report.json" ./...
        $securityExitCode = $LASTEXITCODE
        
        # Also run with human-readable output
        gosec ./... | Tee-Object -FilePath "$ResultsDir/security-tests.log"
        
        return $securityExitCode
    } else {
        Write-TestLog "gosec not installed - skipping security tests" "WARN"
        return 0
    }
}

# Main execution logic
try {
    Write-TestLog "Starting comprehensive test execution: $TestType" "INFO"
    
    # Initialize results directory
    if (Test-Path $ResultsDir) {
        Remove-Item $ResultsDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $ResultsDir -Force | Out-Null
    
    # Start server if needed
    $serverProcess = Start-TestServer
    
    # Execute tests based on type
    $exitCodes = @{}
    
    switch ($TestType.ToLower()) {
        "unit" {
            $exitCodes["unit"] = Invoke-UnitTests
        }
        "integration" {
            $exitCodes["integration"] = Invoke-IntegrationTests
        }
        "e2e" {
            $exitCodes["e2e"] = Invoke-E2ETests
        }
        "security" {
            $exitCodes["security"] = Invoke-SecurityTests
        }
        "all" {
            $exitCodes["unit"] = Invoke-UnitTests
            $exitCodes["integration"] = Invoke-IntegrationTests
            $exitCodes["e2e"] = Invoke-E2ETests
            $exitCodes["security"] = Invoke-SecurityTests
        }
    }
    
    # Generate test summary
    $summary = @{
        Timestamp = Get-Date
        TestType = $TestType
        Results = $exitCodes
        ServerPID = if ($serverProcess) { $serverProcess.Id } else { $null }
    }
    
    $summary | ConvertTo-Json -Depth 3 | Set-Content "$ResultsDir/test-summary.json"
    
    # Determine overall success
    $failures = $exitCodes.Values | Where-Object { $_ -ne 0 }
    
    if ($failures.Count -eq 0) {
        Write-TestLog "All tests completed successfully!" "SUCCESS"
        $overallExitCode = 0
    } else {
        Write-TestLog "Some tests failed. Check individual results for details." "ERROR"
        $overallExitCode = 1
    }
    
    # Coverage report if enabled
    if ($Coverage -and (Test-Path "$CoverageDir/unit-coverage.out")) {
        Write-TestLog "Generating coverage report..." "INFO"
        go tool cover -html="$CoverageDir/unit-coverage.out" -o "$CoverageDir/coverage.html"
        go tool cover -func="$CoverageDir/unit-coverage.out" | Tee-Object -FilePath "$CoverageDir/coverage-summary.txt"
    }
    
    exit $overallExitCode
}
catch {
    Write-TestLog "Test execution failed: $($_.Exception.Message)" "ERROR"
    exit 1
}
finally {
    # Cleanup server process
    if ($serverProcess -and !$serverProcess.HasExited) {
        Write-TestLog "Cleaning up server process..." "INFO"
        Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    }
}
```

### Step 2.2: Database Testing with Testcontainers *(30 minutes)*

#### Enhanced Database Test Setup
Implement proper database testing using Testcontainers for Go, which provides isolated, consistent database environments:

**File**: `internal/testing/database.go`

```go
// Package testing provides test utilities for database operations
package testing

import (
    "context"
    "database/sql"
    "fmt"
    "testing"
    "time"
    
    _ "github.com/mattn/go-sqlite3"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

// DatabaseContainer wraps testcontainers functionality for database testing
type DatabaseContainer struct {
    Container testcontainers.Container
    DB        *sql.DB
    ConnStr   string
}

// NewSQLiteTestDatabase creates an in-memory SQLite database for testing
func NewSQLiteTestDatabase(t *testing.T) *DatabaseContainer {
    t.Helper()
    
    // Create in-memory SQLite database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to create SQLite test database: %v", err)
    }
    
    // Apply migrations/schema
    if err := applyTestSchema(db); err != nil {
        db.Close()
        t.Fatalf("Failed to apply test schema: %v", err)
    }
    
    return &DatabaseContainer{
        Container: nil, // No container for SQLite
        DB:        db,
        ConnStr:   ":memory:",
    }
}

// NewPostgresTestDatabase creates a containerized PostgreSQL database for testing
func NewPostgresTestDatabase(t *testing.T, ctx context.Context) *DatabaseContainer {
    t.Helper()
    
    // Create PostgreSQL container with testcontainers
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForSQL("5432/tcp", "postgres", func(port string) string {
                return fmt.Sprintf("postgres://testuser:testpass@localhost:%s/testdb?sslmode=disable", port)
            }).WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        t.Fatalf("Failed to create PostgreSQL test container: %v", err)
    }
    
    // Get connection string
    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        container.Terminate(ctx)
        t.Fatalf("Failed to get connection string: %v", err)
    }
    
    // Connect to database
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        container.Terminate(ctx)
        t.Fatalf("Failed to connect to PostgreSQL: %v", err)
    }
    
    // Test connection
    if err := db.Ping(); err != nil {
        db.Close()
        container.Terminate(ctx)
        t.Fatalf("Failed to ping PostgreSQL: %v", err)
    }
    
    // Apply test schema
    if err := applyTestSchema(db); err != nil {
        db.Close()
        container.Terminate(ctx)
        t.Fatalf("Failed to apply test schema: %v", err)
    }
    
    return &DatabaseContainer{
        Container: container,
        DB:        db,
        ConnStr:   connStr,
    }
}

// Close cleans up the database resources
func (dc *DatabaseContainer) Close(ctx context.Context) error {
    if dc.DB != nil {
        dc.DB.Close()
    }
    
    if dc.Container != nil {
        return dc.Container.Terminate(ctx)
    }
    
    return nil
}

// SeedTestData inserts test fixtures into the database
func (dc *DatabaseContainer) SeedTestData(t *testing.T) {
    t.Helper()
    
    fixtures := []struct {
        query string
        args  []interface{}
    }{
        {
            "INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
            []interface{}{"Test Article 1", "Test content for article 1", 0.85, time.Now()},
        },
        {
            "INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
            []interface{}{"Test Article 2", "Test content for article 2", 0.72, time.Now()},
        },
        {
            "INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
            []interface{}{"Test Article 3", "Test content for article 3", 0.91, time.Now()},
        },
    }
    
    for _, fixture := range fixtures {
        _, err := dc.DB.Exec(fixture.query, fixture.args...)
        if err != nil {
            t.Fatalf("Failed to seed test data: %v", err)
        }
    }
}

// CleanupTestData removes test data from the database
func (dc *DatabaseContainer) CleanupTestData(t *testing.T) {
    t.Helper()
    
    cleanupQueries := []string{
        "DELETE FROM scores WHERE article_id IN (SELECT id FROM articles WHERE title LIKE 'Test Article%')",
        "DELETE FROM articles WHERE title LIKE 'Test Article%'",
        "DELETE FROM users WHERE email LIKE 'test%@example.com'",
    }
    
    for _, query := range cleanupQueries {
        _, err := dc.DB.Exec(query)
        if err != nil {
            t.Logf("Warning: Failed to cleanup with query '%s': %v", query, err)
        }
    }
}

// applyTestSchema applies the database schema for testing
func applyTestSchema(db *sql.DB) error {
    // Read and apply schema from schema.sql or embedded schema
    schema := `
    CREATE TABLE IF NOT EXISTS articles (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        content TEXT,
        score REAL DEFAULT 0.0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS scores (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        article_id INTEGER NOT NULL,
        score_type TEXT NOT NULL,
        score_value REAL NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (article_id) REFERENCES articles(id)
    );
    
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT UNIQUE NOT NULL,
        name TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `
    
    _, err := db.Exec(schema)
    return err
}

// TestDatabaseWrapper provides common test database functionality
type TestDatabaseWrapper struct {
    *testing.T
    DB      *DatabaseContainer
    Cleanup func()
}

// NewTestDatabase creates a test database with automatic cleanup
func NewTestDatabase(t *testing.T, usePostgres bool) *TestDatabaseWrapper {
    ctx := context.Background()
    
    var db *DatabaseContainer
    if usePostgres {
        db = NewPostgresTestDatabase(t, ctx)
    } else {
        db = NewSQLiteTestDatabase(t)
    }
    
    // Setup automatic cleanup
    cleanup := func() {
        if err := db.Close(ctx); err != nil {
            t.Logf("Warning: Failed to close test database: %v", err)
        }
    }
    
    t.Cleanup(cleanup)
    
    return &TestDatabaseWrapper{
        T:       t,
        DB:      db,
        Cleanup: cleanup,
    }
}
```

#### Example Test Using Database Container

**File**: `internal/api/articles_test.go`

```go
package api

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "your-project/internal/testing" // Adjust import path
)

func TestArticlesEndpoint(t *testing.T) {
    // Create test database
    testDB := testing.NewTestDatabase(t, false) // Use SQLite for fast tests
    
    // Seed test data
    testDB.DB.SeedTestData(t)
    
    // Create handler with test database
    handler := NewArticlesHandler(testDB.DB.DB)
    
    t.Run("GET /articles returns articles", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/articles", nil)
        w := httptest.NewRecorder()
        
        handler.ServeHTTP(w, req)
        
        if w.Code != http.StatusOK {
            t.Errorf("Expected status 200, got %d", w.Code)
        }
        
        // Verify response contains test articles
        body := w.Body.String()
        if !contains(body, "Test Article 1") {
            t.Error("Response should contain test article")
        }
    })
    
    t.Run("POST /articles creates new article", func(t *testing.T) {
        // Test article creation
        // Database will be automatically cleaned up after test
    })
    
    // Database cleanup happens automatically via t.Cleanup()
}

// Helper function for string containment check
func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
        (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
         len(s) > len(substr) && s[1:len(substr)+1] == substr))
}
```

### Step 2.3: Environment Configuration Management *(15 minutes)*

#### Test Environment Configuration

**File**: `configs/test-config.json`

```json
{
  "database": {
    "type": "sqlite",
    "connection": ":memory:",
    "migrations_path": "./migrations",
    "seed_data": true
  },
  "server": {
    "port": 8080,
    "host": "localhost",
    "timeout_seconds": 30,
    "debug_mode": true
  },
  "testing": {
    "parallel_tests": true,
    "cleanup_after_tests": true,
    "coverage_enabled": true,
    "mock_external_services": true
  },
  "logging": {
    "level": "debug",
    "output": "test-results/test.log",
    "format": "json"
  }
}
```

#### Environment Setup Script

**File**: `scripts/setup-test-environment.ps1`

```powershell
# Test Environment Setup Script
param(
    [switch]$Reset,
    [switch]$Verbose
)

function Write-EnvLog {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $(
        switch($Level) {
            "ERROR" { "Red" }
            "WARN" { "Yellow" }
            "SUCCESS" { "Green" }
            default { "White" }
        }
    )
}

try {
    Write-EnvLog "Setting up test environment..." "INFO"
    
    # Create necessary directories
    $dirs = @("test-results", "coverage", "logs", "tmp")
    foreach ($dir in $dirs) {
        if ($Reset -and (Test-Path $dir)) {
            Remove-Item $dir -Recurse -Force
            Write-EnvLog "Cleaned existing directory: $dir" "INFO"
        }
        
        if (!(Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-EnvLog "Created directory: $dir" "SUCCESS"
        }
    }
    
    # Verify Go installation and dependencies
    Write-EnvLog "Verifying Go installation..." "INFO"
    $goVersion = go version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Go is not installed or not in PATH"
    }
    Write-EnvLog "Go version: $goVersion" "SUCCESS"
    
    # Download Go dependencies
    Write-EnvLog "Downloading Go dependencies..." "INFO"
    go mod download
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to download Go dependencies"
    }
    
    # Verify Node.js and npm for E2E tests
    Write-EnvLog "Verifying Node.js installation..." "INFO"
    $nodeVersion = node --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-EnvLog "Node.js version: $nodeVersion" "SUCCESS"
        
        # Install/update npm dependencies
        if (Test-Path "package.json") {
            Write-EnvLog "Installing npm dependencies..." "INFO"
            npm install
            if ($LASTEXITCODE -ne 0) {
                Write-EnvLog "Failed to install npm dependencies" "WARN"
            } else {
                Write-EnvLog "npm dependencies installed successfully" "SUCCESS"
            }
        }
    } else {
        Write-EnvLog "Node.js not found - E2E tests may not work" "WARN"
    }
    
    # Install additional testing tools
    Write-EnvLog "Installing testing tools..." "INFO"
    
    # Install gosec for security testing
    if (!(Get-Command "gosec" -ErrorAction SilentlyContinue)) {
        Write-EnvLog "Installing gosec..." "INFO"
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        if ($LASTEXITCODE -eq 0) {
            Write-EnvLog "gosec installed successfully" "SUCCESS"
        } else {
            Write-EnvLog "Failed to install gosec" "WARN"
        }
    }
    
    # Set up test database
    Write-EnvLog "Setting up test database..." "INFO"
    if (Test-Path "schema.sql") {
        # Create test database and apply schema if needed
        Write-EnvLog "Test database schema available" "SUCCESS"
    }
    
    # Verify Docker for testcontainers (if using PostgreSQL tests)
    Write-EnvLog "Checking Docker availability..." "INFO"
    docker --version 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-EnvLog "Docker is available for testcontainers" "SUCCESS"
    } else {
        Write-EnvLog "Docker not available - using SQLite for database tests" "WARN"
    }
    
    Write-EnvLog "Test environment setup completed successfully!" "SUCCESS"
    Write-EnvLog "Run './scripts/comprehensive-test-runner.ps1' to execute tests" "INFO"
}
catch {
    Write-EnvLog "Test environment setup failed: $($_.Exception.Message)" "ERROR"
    exit 1
}
```

## 📚 Enhanced Testing Infrastructure with Testcontainers

### Key Benefits of Testcontainers Integration

1. **Database Isolation**: Each test gets a fresh database instance
2. **Environment Consistency**: Same database version in development and CI
3. **Parallel Testing**: Tests can run in parallel without conflicts
4. **Easy Cleanup**: Containers are automatically removed after tests

### Database Testing Best Practices

#### Using SQLite for Fast Unit Tests
```go
func TestFastDatabaseOperations(t *testing.T) {
    // Use SQLite for speed in unit tests
    db := testing.NewTestDatabase(t, false) // false = SQLite
    
    // Your test logic here...
}
```

#### Using PostgreSQL for Integration Tests
```go
func TestPostgresIntegration(t *testing.T) {
    // Use PostgreSQL for realistic integration tests
    db := testing.NewTestDatabase(t, true) // true = PostgreSQL
    
    // Test with real PostgreSQL features...
}
```

#### Multi-Database Testing
```go
func TestCrossDatabaseCompatibility(t *testing.T) {
    databases := []struct {
        name        string
        usePostgres bool
    }{
        {"SQLite", false},
        {"PostgreSQL", true},
    }
    
    for _, db := range databases {
        t.Run(db.name, func(t *testing.T) {
            testDB := testing.NewTestDatabase(t, db.usePostgres)
            // Run same test logic against different databases
        })
    }
}
```

## 🎯 Acceptance Criteria

### Phase 2 Completion Checklist
- [ ] **Test Orchestration**: Comprehensive test runner executes all test types
- [ ] **Database Testing**: Testcontainers setup works for both SQLite and PostgreSQL
- [ ] **Environment Management**: Test environment can be reset and configured
- [ ] **Parallel Execution**: Tests can run in parallel without conflicts
- [ ] **Coverage Reporting**: Code coverage is generated and accessible
- [ ] **CI/CD Ready**: Scripts work in both local and CI environments
- [ ] **Error Handling**: Proper cleanup on test failures
- [ ] **Logging**: Comprehensive test execution logging

### Quality Gates
1. **Test Infrastructure Gate**: All test types can be executed successfully
2. **Database Management Gate**: Test databases are created, seeded, and cleaned up properly
3. **Environment Consistency Gate**: Tests produce consistent results across runs
4. **Performance Gate**: Test setup completes within reasonable time (< 2 minutes)

## 🔄 CI/CD Pipeline Preparation

### GitHub Actions Workflow Foundation
Based on GitHub Actions best practices, here's the foundation for CI integration:

**File**: `.github/workflows/test.yml`

```yaml
name: Comprehensive Test Suite
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_USER: testuser  
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
        
    - name: Cache Go modules
      uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        
    - name: Cache npm dependencies
      uses: actions/cache@v4
      with:
        path: ~/.npm
        key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
        
    - name: Install dependencies
      run: |
        go mod download
        npm install
        
    - name: Setup test environment
      run: |
        chmod +x ./scripts/setup-test-environment.sh
        ./scripts/setup-test-environment.sh
        
    - name: Run comprehensive tests
      run: |
        chmod +x ./scripts/comprehensive-test-runner.sh
        ./scripts/comprehensive-test-runner.sh --coverage
        
    - name: Upload test results
      uses: actions/upload-artifact@v4
      if: always()
      with:
        name: test-results
        path: test-results/
        
    - name: Upload coverage reports
      uses: actions/upload-artifact@v4
      if: always()
      with:
        name: coverage-reports
        path: coverage/
```

## 🔍 Validation Commands

### Test Phase 2 Success
Run these commands to validate Phase 2 completion:

```powershell
# 1. Setup test environment
.\scripts\setup-test-environment.ps1 -Verbose
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Environment Setup: PASS" -ForegroundColor Green } else { Write-Host "❌ Environment Setup: FAIL" -ForegroundColor Red }

# 2. Run unit tests with database
.\scripts\comprehensive-test-runner.ps1 -TestType unit -Coverage
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Unit Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Unit Tests: FAIL" -ForegroundColor Red }

# 3. Test database containers (if Docker available)
go test -tags=integration ./internal/testing/...
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Database Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Database Tests: FAIL" -ForegroundColor Red }

# 4. Verify test artifacts
if (Test-Path "test-results/test-summary.json") { Write-Host "✅ Test Artifacts: PASS" -ForegroundColor Green } else { Write-Host "❌ Test Artifacts: FAIL" -ForegroundColor Red }
```

## 🚫 Rollback Procedures

### If Test Infrastructure Fails
```powershell
# Rollback script: rollback-phase2-infrastructure.ps1
Write-Host "🔄 Rolling back Phase 2 test infrastructure..." -ForegroundColor Yellow

# Stop any running test processes
Get-Process | Where-Object { $_.ProcessName -eq "go" } | Stop-Process -Force -ErrorAction SilentlyContinue

# Clean up test artifacts
$cleanupDirs = @("test-results", "coverage", "tmp")
foreach ($dir in $cleanupDirs) {
    if (Test-Path $dir) {
        Remove-Item $dir -Recurse -Force
        Write-Host "  ✅ Cleaned directory: $dir" -ForegroundColor Green
    }
}

# Remove test configuration if problematic
if (Test-Path "configs/test-config.json.backup") {
    Move-Item "configs/test-config.json.backup" "configs/test-config.json" -Force
    Write-Host "  ✅ Restored configuration backup" -ForegroundColor Green
}

Write-Host "🔄 Phase 2 rollback complete" -ForegroundColor Green
```

## 📞 Support & Escalation

**If you encounter issues during Phase 2:**

1. **Database Issues**: Check Docker installation, container permissions
2. **Test Runner Issues**: Verify PowerShell execution policy, Go installation
3. **Environment Issues**: Check file permissions, disk space
4. **Dependency Issues**: Verify internet connectivity, proxy settings

**Common Solutions**:
- **Port Conflicts**: Use `netstat -an | findstr :8080` to check port usage
- **Permission Errors**: Ensure PowerShell execution policy allows scripts
- **Docker Issues**: Verify Docker Desktop is running and accessible

---

**Phase 2 Status**: ⏳ READY FOR IMPLEMENTATION  
**Next Phase**: `04_phase3_e2e_stabilization.md`  
**Estimated Total Time**: 1-2 hours  
**Key Dependencies**: Docker (optional), Go 1.21+, Node.js 18+ (for E2E)
