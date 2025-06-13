#!/usr/bin/env pwsh

Write-Host "Testing NBG API Endpoints" -ForegroundColor Green
Write-Host "=========================" -ForegroundColor Green

# Test health endpoint
Write-Host "`nTesting health endpoint..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/healthz" -Method Get
    Write-Host "✓ Health check passed: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "✗ Health check failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test articles endpoint
Write-Host "`nTesting articles endpoint..." -ForegroundColor Yellow
try {
    $articles = Invoke-RestMethod -Uri "http://localhost:8080/api/articles" -Method Get
    Write-Host "✓ Articles endpoint working" -ForegroundColor Green
    Write-Host "  - Success: $($articles.success)" -ForegroundColor Cyan
    Write-Host "  - Articles returned: $($articles.data.Count)" -ForegroundColor Cyan
    
    if ($articles.data.Count -gt 0) {
        $firstArticle = $articles.data[0]
        Write-Host "  - First article ID: $($firstArticle.article_id)" -ForegroundColor Cyan
        Write-Host "  - First article source: $($firstArticle.source)" -ForegroundColor Cyan
        
        # Test individual article endpoint
        Write-Host "`nTesting individual article endpoint..." -ForegroundColor Yellow
        try {
            $article = Invoke-RestMethod -Uri "http://localhost:8080/api/articles/$($firstArticle.article_id)" -Method Get
            Write-Host "✓ Individual article endpoint working" -ForegroundColor Green
            Write-Host "  - Article title: $($article.data.title)" -ForegroundColor Cyan
        } catch {
            Write-Host "✗ Individual article endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
} catch {
    Write-Host "✗ Articles endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`nAPI testing complete!" -ForegroundColor Green
