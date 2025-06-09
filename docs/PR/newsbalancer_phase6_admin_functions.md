# NewsBalancer Phase 6: Admin Functions

## Admin-Only Functions

The NewsBalancer system includes several administrative functions that enable monitoring, management, and manual intervention when needed. These functions are accessible through dedicated API endpoints and the admin dashboard:

- **Manual Scoring:** `POST /api/manual-score/:id` with `{"score": float}` to override a bias value.  
- **RSS Refresh:** `POST /api/refresh` to trigger background feed collection.  
- **System Metrics / Health:** `/metrics/*` and `/healthz` endpoints provide JSON data used by admin dashboard charts or tiles.

These functions give administrators direct control over the system when needed, while providing visibility into system performance and health.

## 🔄 Admin Functions Verification

```powershell
# phase6_admin_functions.ps1 - Validates admin functionality
function Test-AdminFunctions {
    Write-Host "🔍 Testing admin functionality..."
    
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
        
        # Test manual scoring endpoint
        $manualScoreResult = Test-ManualScoring $articleId
        
        # Test RSS refresh endpoint
        $rssRefreshResult = Test-RSSRefresh
        
        # Test metrics/health endpoints
        $metricsResult = Test-MetricsEndpoints
        
        # Test admin dashboard
        $dashboardResult = Test-AdminDashboard
        
        $success = (
            $manualScoreResult.success -and
            $rssRefreshResult.success -and
            $metricsResult.success -and
            $dashboardResult.success
        )
        
        return @{
            success = $success
            articleId = $articleId
            details = @{
                manualScoring = $manualScoreResult
                rssRefresh = $rssRefreshResult
                metrics = $metricsResult
                dashboard = $dashboardResult
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

function Test-ManualScoring {
    param([string]$articleId)
    
    try {
        # Get current score
        $initialScore = Get-ArticleScore $articleId
        
        # Test manual score API endpoint
        $newScore = if ($initialScore -ge 0) { -0.5 } else { 0.5 }
        $body = @{
            score = $newScore
        } | ConvertTo-Json
        
        $headers = @{
            "Content-Type" = "application/json"
        }
        
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/manual-score/$articleId" -Method Post -Body $body -Headers $headers -UseBasicParsing
        
        # Verify score was updated
        Start-Sleep -Seconds 2
        $updatedScore = Get-ArticleScore $articleId
        $scoreChanged = [Math]::Abs($updatedScore - $initialScore) -gt 0.1
        
        return @{
            success = $response.StatusCode -eq 200 -and $scoreChanged
            statusCode = $response.StatusCode
            initialScore = $initialScore
            targetScore = $newScore
            updatedScore = $updatedScore
            scoreChanged = $scoreChanged
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Get-ArticleScore {
    param([string]$articleId)
    
    try {
        # Get the article's current score
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/articles/$articleId/bias" -UseBasicParsing
        $content = $response.Content | ConvertFrom-Json
        
        return $content.composite_score
    }
    catch {
        return $null
    }
}

function Test-RSSRefresh {
    try {
        # Test RSS refresh API endpoint
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/refresh" -Method Post -UseBasicParsing
        
        return @{
            success = $response.StatusCode -eq 202
            statusCode = $response.StatusCode
        }
    }
    catch {
        # Check if we got 202 but error parsing response
        if ($_.Exception.Response.StatusCode -eq 202) {
            return @{
                success = $true
                statusCode = 202
                note = "Got 202 status but could not parse response"
            }
        }
        
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-MetricsEndpoints {
    try {
        # Test metrics/health endpoints
        $healthzResult = Invoke-WebRequest -Uri "http://localhost:8080/healthz" -UseBasicParsing
        $metricsResult = Invoke-WebRequest -Uri "http://localhost:8080/metrics" -UseBasicParsing
        
        # Check health endpoint
        $healthSuccess = $healthzResult.StatusCode -eq 200
        
        # Check metrics endpoint
        $metricsSuccess = $metricsResult.StatusCode -eq 200
        $hasMetricsData = $metricsResult.Content -match 'newsbalancer'
        
        return @{
            success = $healthSuccess -and $metricsSuccess -and $hasMetricsData
            healthzStatus = $healthzResult.StatusCode
            metricsStatus = $metricsResult.StatusCode
            hasMetricsData = $hasMetricsData
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-AdminDashboard {
    try {
        # Test admin dashboard
        $response = Invoke-WebRequest -Uri "http://localhost:8080/admin" -UseBasicParsing
        
        # Check for expected elements
        $hasDashboard = $response.Content -match '<div\s+class=[''"]admin-dashboard[''"]'
        $hasMetricsDisplay = $response.Content -match '<div\s+class=[''"]metrics-display[''"]'
        $hasControlPanel = $response.Content -match '<div\s+class=[''"]control-panel[''"]'
        
        return @{
            success = $hasDashboard -and $hasMetricsDisplay -and $hasControlPanel
            statusCode = $response.StatusCode
            hasDashboard = $hasDashboard
            hasMetricsDisplay = $hasMetricsDisplay
            hasControlPanel = $hasControlPanel
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
$controller.UpdatePhase("admin-functions")

$result = Test-AdminFunctions
if ($result.success) {
    $controller.CompletePhase("admin-functions")
    $controller.CurrentState.componentsStatus["adminFunctions"] = "complete"
    $controller.SaveState()
    
    Write-Host "✅ Admin functions tests passed successfully" -ForegroundColor Green
    
    # Generate final implementation report
    $controller.GenerateReport("frontend_implementation_report.json")
    
    Write-Host "🎉 All frontend implementation phases completed successfully!" -ForegroundColor Green
} else {
    Write-Host "❌ Admin functions tests failed" -ForegroundColor Red
    
    # Generate recommendations based on what failed
    $recommendations = @()
    if (-not $result.details.manualScoring.success) {
        $recommendations += "Fix manual scoring endpoint (/api/manual-score/:id) - POST request not working or score not updated"
    }
    if (-not $result.details.rssRefresh.success) {
        $recommendations += "Fix RSS refresh endpoint (/api/refresh) - should return 202 Accepted"
    }
    if (-not $result.details.metrics.success) {
        $recommendations += "Fix metrics/health endpoints - /healthz or /metrics not responding properly"
    }
    if (-not $result.details.dashboard.success) {
        $recommendations += "Fix admin dashboard template - missing required elements"
    }
    
    foreach ($rec in $recommendations) {
        Write-Host "  - $rec" -ForegroundColor Yellow
    }
    
    $controller.LogIssue("adminFunctions", "Admin functions failed", "major")
}
```

## Implementation Details

### Manual Scoring Endpoint

The manual scoring endpoint allows administrators to override the automatically calculated bias score for an article. This is useful for:
- Correcting obviously incorrect AI assessments
- Setting a baseline for controversial articles
- Testing the UI with specific bias values

Implementation details:
- HTTP Method: POST
- URL: `/api/manual-score/:id`
- Request body: `{"score": float}`
- Response: 200 OK on success

```go
// Pseudo-code for manual scoring endpoint
func ManualScoreHandler(w http.ResponseWriter, r *http.Request) {
    // Extract article ID from URL
    articleID := getArticleIDFromRequest(r)
    
    // Parse request body
    var request struct {
        Score float64 `json:"score"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }
    
    // Validate score range
    if request.Score < -1.0 || request.Score > 1.0 {
        http.Error(w, "Score must be between -1.0 and 1.0", http.StatusBadRequest)
        return
    }
    
    // Update article score in database
    err := updateArticleScore(articleID, request.Score, true) // true = manually set
    if err != nil {
        http.Error(w, "Failed to update article score", http.StatusInternalServerError)
        return
    }
    
    // Return success response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "message": "Article score updated successfully"
    })
}
```

### RSS Refresh Endpoint

The RSS refresh endpoint triggers a background job to collect new articles from configured RSS feeds. This allows administrators to:
- Force an immediate refresh of content
- Test feed configuration changes
- Ensure the latest news is available without waiting for scheduled jobs

Implementation details:
- HTTP Method: POST
- URL: `/api/refresh`
- Response: 202 Accepted (async operation)

```go
// Pseudo-code for RSS refresh endpoint
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
    // Trigger background refresh job
    go func() {
        refreshFeeds()
    }()
    
    // Return accepted response
    w.WriteHeader(http.StatusAccepted)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "accepted",
        "message": "RSS refresh job started"
    })
}
```

### Metrics and Health Endpoints

The metrics and health endpoints provide system monitoring data:

- `/healthz` - Simple health check endpoint returning 200 OK if the system is operational
- `/metrics` - Detailed system metrics in a format compatible with monitoring systems

These endpoints support:
- External monitoring and alerting
- Dashboard visualizations
- Performance tracking

The admin dashboard uses this data to display:
- Article processing rates
- Error counts
- Database statistics
- Cache hit rates
- API response times

### Admin Dashboard

The admin dashboard (`/admin`) provides a central interface for system management with:

1. **System Metrics Display**
   - Charts showing article processing rates
   - Bias distribution visualization
   - Error rate monitoring
   - Resource utilization metrics

2. **Control Panel**
   - RSS refresh button
   - Cache clear options
   - System configuration toggles

3. **Administrative Tools**
   - Article search and manual scoring interface
   - Feed management options
   - User feedback review

The dashboard is implemented as a server-rendered template with JavaScript for interactive components.

## Verification Process

The automated verification script tests:
1. Manual scoring functionality with score change verification
2. RSS refresh endpoint with proper 202 Accepted response
3. Metrics and health endpoints with valid responses
4. Admin dashboard template with required elements

Each test verifies both the correct HTTP status code and the expected behavior (e.g., score change after manual scoring), ensuring that all administrative functions work correctly.
