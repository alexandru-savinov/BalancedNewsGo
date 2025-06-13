#!/usr/bin/env pwsh
# Test script for HTMX endpoints

Write-Host "=== Testing HTMX Fragment Endpoints ===" -ForegroundColor Green

$baseUrl = "http://localhost:8080"
$endpoints = @(
    @{ Name = "Articles Fragment"; Url = "$baseUrl/api/fragments/articles" },
    @{ Name = "Article Detail Fragment"; Url = "$baseUrl/api/fragments/article/139" },
    @{ Name = "Article Summary Fragment"; Url = "$baseUrl/api/fragments/article/139/summary" },
    @{ Name = "Main Articles Page"; Url = "$baseUrl/articles" },
    @{ Name = "API Articles Endpoint"; Url = "$baseUrl/api/articles" }
)

foreach ($endpoint in $endpoints) {
    Write-Host "`nTesting: $($endpoint.Name)" -ForegroundColor Yellow
    Write-Host "URL: $($endpoint.Url)"
    
    try {
        $response = Invoke-WebRequest -Uri $endpoint.Url -Method GET -TimeoutSec 10
        Write-Host "✅ Status: $($response.StatusCode)" -ForegroundColor Green
        
        # Show content type and length
        $contentType = $response.Headers["Content-Type"]
        Write-Host "Content-Type: $contentType"
        Write-Host "Content-Length: $($response.Content.Length) characters"
        
        # Show first few characters for HTML responses
        if ($contentType -like "*html*") {
            $preview = $response.Content.Substring(0, [Math]::Min(100, $response.Content.Length))
            Write-Host "Preview: $($preview.Replace("`n", " ").Replace("`r", ""))" -ForegroundColor Cyan
        }
        
        # For JSON responses, show structure
        if ($contentType -like "*json*") {
            $json = $response.Content | ConvertFrom-Json
            if ($json.data) {
                Write-Host "Data items: $($json.data.Length)" -ForegroundColor Cyan
            }
        }
        
    } catch {
        Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host "`n=== Test Complete ===" -ForegroundColor Green
