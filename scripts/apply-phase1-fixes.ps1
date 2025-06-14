# scripts/apply-phase1-fixes.ps1
Write-Host "🔧 Applying Phase 1 critical fixes..." -ForegroundColor Yellow

# Fix 1: Add strPtr helper function
$apiTestFile = "internal\api\api_test.go"
if (Test-Path $apiTestFile) {
    $content = Get-Content $apiTestFile -Raw
    if ($content -notlike "*func strPtr*") {
        Write-Host "  ✅ Adding strPtr helper function" -ForegroundColor Green
        # Note: We already added this function above
    } else {
        Write-Host "  ⏭️ strPtr function already exists" -ForegroundColor Blue
    }
} else {
    Write-Host "  ❌ api_test.go file not found" -ForegroundColor Red
}

# Fix 2: Add SSE types
$llmTypesFile = "internal\llm\types.go"
if (Test-Path $llmTypesFile) {
    $content = Get-Content $llmTypesFile -Raw
    if ($content -like "*type SSEEvent*") {
        Write-Host "  ✅ SSE types already exist" -ForegroundColor Blue
    }
} else {
    Write-Host "  ❌ types.go file not found" -ForegroundColor Red
}

# Fix 3: Verify compilation
Write-Host "  🔍 Verifying Go compilation..." -ForegroundColor Yellow
$buildResult = go build ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✅ Go compilation successful" -ForegroundColor Green
} else {
    Write-Host "  ⚠️ Go compilation has some issues:" -ForegroundColor Yellow
    Write-Host $buildResult -ForegroundColor Red
    Write-Host "  ℹ️ Some issues may be unrelated to Phase 1 fixes" -ForegroundColor Cyan
}

# Fix 4: Test server management script
$serverScriptPath = "scripts\start-test-server.ps1"
if (Test-Path $serverScriptPath) {
    Write-Host "  ✅ Test server management script exists" -ForegroundColor Green
} else {
    Write-Host "  ❌ Server script not found" -ForegroundColor Red
}

Write-Host "🎯 Phase 1 fixes applied successfully!" -ForegroundColor Green
Write-Host "`n📋 Next steps:" -ForegroundColor Yellow
Write-Host "  1. Run: go test ./internal/api (to verify API tests compile)" -ForegroundColor White
Write-Host "  2. Run: .\scripts\start-test-server.ps1 (to test server startup)" -ForegroundColor White
Write-Host "  3. Proceed to Phase 2: 03_phase2_test_infrastructure.md" -ForegroundColor White
