#!/usr/bin/env pwsh
# Emergency Build Validation Script
# Tests all critical components to verify recovery

Write-Host "=== EMERGENCY BUILD VALIDATION ===" -ForegroundColor Yellow
Write-Host "Timestamp: $(Get-Date)" -ForegroundColor Gray
Write-Host "Git branch: $(git branch --show-current)" -ForegroundColor Gray
Write-Host ""

# Define test matrix
$components = @{
    "cmd/server" = "Core server functionality"
    "cmd/score_articles" = "Article scoring CLI"
    "cmd/test_handlers" = "Template handler testing"
    "cmd/test_reanalyze" = "Article reanalysis CLI"
    "internal/api" = "API layer"
    "internal/llm" = "LLM integration"
    "internal/models" = "Data models"
    "internal/db" = "Database layer"
}

$passed = 0
$failed = 0
$total = $components.Count

foreach ($component in $components.Keys) {
    $description = $components[$component]
    Write-Host "Testing $component ($description)..." -ForegroundColor Cyan
    
    try {
        $result = go build ./$component 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ $component - BUILD SUCCESS" -ForegroundColor Green
            $passed++
        } else {
            Write-Host "❌ $component - BUILD FAILED" -ForegroundColor Red
            Write-Host "Error details:" -ForegroundColor Red
            Write-Host $result -ForegroundColor Red
            $failed++
        }
    } catch {
        Write-Host "❌ $component - BUILD FAILED (Exception)" -ForegroundColor Red
        Write-Host $_.Exception.Message -ForegroundColor Red
        $failed++
    }
}

# Calculate metrics
$successRate = [math]::Round(($passed * 100) / $total, 1)

Write-Host ""
Write-Host "=== EMERGENCY VALIDATION SUMMARY ===" -ForegroundColor Yellow
Write-Host "Passed: $passed/$total" -ForegroundColor Green
Write-Host "Failed: $failed/$total" -ForegroundColor Red
Write-Host "Success Rate: $successRate%" -ForegroundColor $(if ($successRate -ge 90) { "Green" } elseif ($successRate -ge 70) { "Yellow" } else { "Red" })

if ($failed -eq 0) {
    Write-Host "🎉 EMERGENCY RECOVERY SUCCESSFUL!" -ForegroundColor Green
    Write-Host "All components building - ready for functional testing" -ForegroundColor Green
    exit 0
} else {
    Write-Host "⚠️ EMERGENCY RECOVERY INCOMPLETE" -ForegroundColor Yellow
    Write-Host "$failed components still failing - requires attention" -ForegroundColor Yellow
    exit 1
}
