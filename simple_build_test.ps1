#!/usr/bin/env pwsh
# Simple Build Test Script

Write-Host "=== SIMPLE BUILD TEST ===" -ForegroundColor Green
Write-Host "Testing critical builds..."

# Test 1: Server
Write-Host "Testing server build..." -NoNewline
try {
    $null = go build ./cmd/server 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host " ✅ PASS" -ForegroundColor Green
    } else {
        Write-Host " ❌ FAIL" -ForegroundColor Red
    }
} catch {
    Write-Host " ❌ ERROR" -ForegroundColor Red
}

# Test 2: API
Write-Host "Testing API build..." -NoNewline
try {
    $null = go build ./internal/api 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host " ✅ PASS" -ForegroundColor Green
    } else {
        Write-Host " ❌ FAIL" -ForegroundColor Red
    }
} catch {
    Write-Host " ❌ ERROR" -ForegroundColor Red
}

# Test 3: Score Articles CLI
Write-Host "Testing score_articles build..." -NoNewline
try {
    $null = go build ./cmd/score_articles 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host " ✅ PASS" -ForegroundColor Green
    } else {
        Write-Host " ❌ FAIL" -ForegroundColor Red
    }
} catch {
    Write-Host " ❌ ERROR" -ForegroundColor Red
}

Write-Host "Test completed!" -ForegroundColor Yellow
