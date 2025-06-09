# NewsBalancer Phase 4: Bias Analysis Data APIs

## Bias Analysis Data APIs

The NewsBalancer frontend requires access to detailed bias analysis data through dedicated API endpoints. These endpoints provide JSON data for programmatic use and client-side rendering:

- **`GET /api/articles/:id/bias`** – Returns JSON with `composite_score` and `results` (per-model scores).  
- **`GET /api/articles/:id/ensemble`** – Returns detailed ensemble breakdown including sub_results and aggregation stats.

Both endpoints are cached for performance and reflect the latest stored analysis data.

## 🔄 API Endpoints Verification

```powershell
# phase4_api_verification.ps1 - Validates bias analysis API endpoints
function Test-BiasAnalysisAPIs {
    Write-Host "🔍 Testing bias analysis API endpoints..."
    
    # Start the server if not running
    $serverJob = Start-Process -FilePath "make" -ArgumentList "run" -NoNewWindow -PassThru
    Start-Sleep -Seconds 5  # Allow server to start
    
    try {
        # First, get an article ID to test with
        $articleId = Get-TestArticleId
        
        if (-not $articleId) {
            return @{
                success = $false
                error = "Could not find a valid article ID for testing"
            }
        }
        
        # Test the bias API endpoint
        $biasResult = Test-BiasEndpoint $articleId
        
        # Test the ensemble API endpoint
        $ensembleResult = Test-EnsembleEndpoint $articleId
        
        # Test caching by measuring response times
        $cachingResult = Test-APICaching $articleId
        
        $success = (
            $biasResult.success -and
            $ensembleResult.success -and
            $cachingResult.success
        )
        
        return @{
            success = $success
            articleId = $articleId
            details = @{
                biasEndpoint = $biasResult
                ensembleEndpoint = $ensembleResult
                caching = $cachingResult
            }
        }
    }
    finally {
        # Cleanup - stop the server
        if ($serverJob -ne $null) {
            Stop-Process -Id $serverJob.Id -Force -ErrorAction SilentlyContinue
        }
    }
}

function Get-TestArticleId {
    try {
        # Get articles listing and extract first article ID
        $response = Invoke-WebRequest -Uri "http://localhost:8080/articles" -UseBasicParsing
        
        # Try to extract article ID from links
        if ($response.Content -match '/article/(\d+)') {
            return $matches[1]
        }
        
        return $null
    }
    catch {
        return $null
    }
}

function Test-BiasEndpoint {
    param([string]$articleId)
    
    try {
        # Test bias API endpoint
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/articles/$articleId/bias" -UseBasicParsing
        
        # Parse response as JSON
        $content = $response.Content | ConvertFrom-Json
        
        # Check for expected fields
        $hasCompositeScore = $content.composite_score -ne $null
        $hasResults = $content.results -ne $null
        
        return @{
            success = $hasCompositeScore -and $hasResults
            statusCode = $response.StatusCode
            hasCompositeScore = $hasCompositeScore
            hasResults = $hasResults
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-EnsembleEndpoint {
    param([string]$articleId)
    
    try {
        # Test ensemble API endpoint
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/articles/$articleId/ensemble" -UseBasicParsing
        
        # Parse response as JSON
        $content = $response.Content | ConvertFrom-Json
        
        # Check for expected fields
        $hasSubResults = $content.sub_results -ne $null
        $hasAggregation = $content.aggregation -ne $null
        
        return @{
            success = $hasSubResults -and $hasAggregation
            statusCode = $response.StatusCode
            hasSubResults = $hasSubResults
            hasAggregation = $hasAggregation
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-APICaching {
    param([string]$articleId)
    
    try {
        # Test API caching by measuring response times
        $times = @()
        
        # Make several requests and measure times
        for ($i = 0; $i -lt 3; $i++) {
            $start = Get-Date
            Invoke-WebRequest -Uri "http://localhost:8080/api/articles/$articleId/bias" -UseBasicParsing | Out-Null
            $end = Get-Date
            $times += ($end - $start).TotalMilliseconds
        }
        
        # The first request is typically slower than subsequent ones if caching works
        $firstRequest = $times[0]
        $subsequentAvg = ($times[1..2] | Measure-Object -Average).Average
        
        # If subsequent requests are significantly faster, caching likely works
        $cachingWorking = $subsequentAvg -lt ($firstRequest * 0.8)
        
        return @{
            success = $cachingWorking
            firstRequestTime = $firstRequest
            subsequentAvgTime = $subsequentAvg
            improvement = if ($firstRequest -gt 0) { [math]::Round(($firstRequest - $subsequentAvg) / $firstRequest * 100, 1) } else { 0 }
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

# Integration with feedback controller
$controller = [FeedbackController]::new()
$controller.Initialize()
$controller.UpdatePhase("bias-apis")

$result = Test-BiasAnalysisAPIs
if ($result.success) {
    $controller.CompletePhase("bias-apis")
    $controller.CurrentState.componentsStatus["biasAPIs"] = "complete"
    $controller.SaveState()
    
    Write-Host "✅ Bias analysis API tests passed successfully" -ForegroundColor Green
    if ($result.details.caching.improvement -gt 20) {
        Write-Host "  📈 API caching is working well (${$result.details.caching.improvement}% faster subsequent requests)" -ForegroundColor Green
    }
} else {
    Write-Host "❌ Bias analysis API tests failed" -ForegroundColor Red
    
    # Generate recommendations based on what failed
    $recommendations = @()
    if (-not $result.details.biasEndpoint.success) {
        $recommendations += "Fix bias API endpoint (/api/articles/:id/bias) - missing composite_score or results fields"
    }
    if (-not $result.details.ensembleEndpoint.success) {
        $recommendations += "Fix ensemble API endpoint (/api/articles/:id/ensemble) - missing sub_results or aggregation fields"
    }
    if (-not $result.details.caching.success) {
        $recommendations += "Implement or fix API response caching - subsequent requests should be faster"
    }
    
    foreach ($rec in $recommendations) {
        Write-Host "  - $rec" -ForegroundColor Yellow
    }
    
    $controller.LogIssue("biasAPIs", "Bias analysis API endpoints failed tests", "major")
}
```

## Implementation Details

### Bias API Endpoint

The `/api/articles/:id/bias` endpoint should return a JSON response with the following structure:

```json
{
  "composite_score": 0.25,
  "confidence": 0.85,
  "bias_label": "Right Leaning",
  "results": [
    {
      "model": "gpt-4",
      "score": 0.3,
      "confidence": 0.9
    },
    {
      "model": "claude-2",
      "score": 0.2,
      "confidence": 0.8
    }
  ]
}
```

The endpoint provides a summary of the bias analysis for an article, including:
- `composite_score`: The overall bias score between -1 (left) and +1 (right)
- `confidence`: How confident the system is in the bias assessment (0-1)
- `bias_label`: A human-readable label derived from the score
- `results`: Array of individual model scores that contributed to the composite

### Ensemble API Endpoint

The `/api/articles/:id/ensemble` endpoint provides a more detailed breakdown of the bias analysis process with the following structure:

```json
{
  "article_id": 123,
  "composite_score": 0.25,
  "confidence": 0.85,
  "sub_results": [
    {
      "model": "gpt-4",
      "score": 0.3,
      "confidence": 0.9,
      "reasoning": "The article consistently uses language that...",
      "analysis_time": "2023-06-15T14:30:45Z"
    },
    {
      "model": "claude-2",
      "score": 0.2,
      "confidence": 0.8,
      "reasoning": "Based on the framing of economic issues...",
      "analysis_time": "2023-06-15T14:31:12Z"
    }
  ],
  "aggregation": {
    "method": "weighted_average",
    "weights": {
      "gpt-4": 0.6,
      "claude-2": 0.4
    },
    "variance": 0.05
  }
}
```

This endpoint provides:
- Detailed sub-results from each model, including reasoning
- Information about how the composite score was aggregated
- Timestamps for each analysis
- Statistical information like variance between models

### API Caching Implementation

Both API endpoints should implement efficient caching to improve performance:

```go
// Pseudo-code for API caching in Go
func BiasHandler(w http.ResponseWriter, r *http.Request) {
    articleID := getArticleIDFromRequest(r)
    
    // Check cache first
    cacheKey := fmt.Sprintf("bias-%d", articleID)
    if cachedData, found := cache.Get(cacheKey); found {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Cache", "HIT")
        w.Write(cachedData.([]byte))
        return
    }
    
    // If not in cache, get from database
    bias, err := getBiasFromDB(articleID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Convert to JSON
    jsonData, err := json.Marshal(bias)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Store in cache
    cache.Set(cacheKey, jsonData, 5*time.Minute)
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Cache", "MISS")
    w.Write(jsonData)
}
```

The caching system should:
1. Use a suitable in-memory cache (like go-cache)
2. Set appropriate expiration times (e.g., 5 minutes)
3. Invalidate cache entries when bias scores are updated
4. Include cache status headers for debugging

## Verification Process

The automated verification script tests:
1. Bias API endpoint response format and required fields
2. Ensemble API endpoint response format and required fields
3. API caching effectiveness by measuring response times

The caching test specifically measures the time difference between the first request and subsequent requests, ensuring that cached responses are significantly faster, which indicates proper caching implementation.
