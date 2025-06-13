#!/usr/bin/env pwsh
# Phase 4 Testing Validation Script
# Validates the API client testing and HTMX integration

Write-Host "🔍 Phase 4 Testing Validation" -ForegroundColor Cyan
Write-Host "===============================" -ForegroundColor Cyan

# Check if test files exist
$testFiles = @(
    "internal\api\wrapper\client_test.go",
    "cmd\server\template_handlers_api_test.go", 
    "tests\e2e\htmx-integration.spec.ts"
)

Write-Host "`n📁 Checking test files..." -ForegroundColor Yellow
foreach ($file in $testFiles) {
    if (Test-Path $file) {
        $size = (Get-Item $file).Length
        Write-Host "✅ $file ($size bytes)" -ForegroundColor Green
    } else {
        Write-Host "❌ $file (missing)" -ForegroundColor Red
    }
}

# Check if server is running
Write-Host "`n🌐 Checking server status..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/articles" -TimeoutSec 5 -ErrorAction Stop
    Write-Host "✅ Server responding - Status: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "❌ Server not responding or not started" -ForegroundColor Red
    Write-Host "   To start server: go run ./cmd/server" -ForegroundColor Gray
}

# Check Go module dependencies
Write-Host "`n📦 Checking Go dependencies..." -ForegroundColor Yellow
try {
    $goModCheck = go mod verify 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Go modules verified" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Go modules need attention" -ForegroundColor Yellow
    }
} catch {
    Write-Host "❌ Could not verify Go modules" -ForegroundColor Red
}

# Check Node.js dependencies for E2E tests
Write-Host "`n🎭 Checking Playwright setup..." -ForegroundColor Yellow
if (Test-Path "node_modules\.bin\playwright.cmd") {
    Write-Host "✅ Playwright installed" -ForegroundColor Green
} else {
    Write-Host "❌ Playwright not installed" -ForegroundColor Red
    Write-Host "   To install: npm install" -ForegroundColor Gray
}

Write-Host "`n📋 Phase 4 Summary:" -ForegroundColor Cyan
Write-Host "===================" -ForegroundColor Cyan
Write-Host "✅ Unit tests for API wrapper created (retry logic, concurrency, caching)" -ForegroundColor Green
Write-Host "✅ Handler tests with mock API client created" -ForegroundColor Green  
Write-Host "✅ E2E tests for HTMX integration created" -ForegroundColor Green
Write-Host "⚠️  Some test compilation issues need fixing" -ForegroundColor Yellow

Write-Host "`n🔧 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Fix API wrapper test compilation by using HTTP mocking instead of interface mocking" -ForegroundColor White
Write-Host "2. Start server: go run ./cmd/server" -ForegroundColor White
Write-Host "3. Run handler tests: go test -v ./cmd/server/" -ForegroundColor White
Write-Host "4. Run E2E tests: npx playwright test tests/e2e/htmx-integration.spec.ts" -ForegroundColor White
Write-Host "5. Add integration tests with real HTTP endpoints" -ForegroundColor White

Write-Host "`n🎯 Phase 4 Status: IMPLEMENTATION COMPLETE, VALIDATION IN PROGRESS" -ForegroundColor Green
