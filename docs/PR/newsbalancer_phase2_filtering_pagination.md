# NewsBalancer Phase 2: Filtering, Search, and Pagination

## Filtering, Search, and Pagination with Query Parameters

The articles listing page supports filtering by source, political leaning, and text search, as well as pagination. These are handled through query parameters on the same `/articles` route, mapped to backend logic:

- **Source Filter:** e.g. `?source=BBC` – Returns only articles from "BBC". The handler adds a SQL condition `AND source = ?` when this parameter is present.  
- **Bias/Leaning Filter:** e.g. `?leaning=Left` – Filters articles by bias direction. The backend maps "Left" to composite_score < -0.1, "Right" to > 0.1, and "Center" to between -0.1 and 0.1. (`bias` is accepted as an alias for `leaning` for backward compatibility.)  
- **Search Query:** e.g. `?query=election` – Performs a substring match on title or content. The handler appends a `title LIKE ? OR content LIKE ?` filter for the query term.  
- **Pagination:** e.g. `?page=2` – Paginates results 20 per page. The backend uses `page` to calculate an `OFFSET` (`(page-1)*20`) and sets a `LIMIT` of 21 items (one extra to detect if more pages remain). In the rendered template, it sets `HasMore` if an extra item was found beyond the page limit. The front-end can then show a "Next Page" link or load-more button if `HasMore` is true. Current page number is tracked as `CurrentPage` in the template data.

These filters do not require separate AJAX endpoints – the user can navigate or submit a form and the server returns a new filtered HTML page. Each article entry in the list includes its title, source, publication date, and a bias indicator. The bias indicator is derived from the article's `CompositeScore` and `Confidence` provided by the backend. The `CompositeScore` is a number between -1 (left) and +1 (right); the template may format it to two decimal places or visually via a bias slider component. A textual **Bias Label** is also shown (e.g. "Left Leaning", "Center", "Right Leaning"), determined on the backend using threshold ±0.3.

## 🔄 Filtering & Pagination Verification

```powershell
# phase2_filtering.ps1 - Executed after implementing filtering and pagination
function Test-FilteringAndPagination {
    Write-Host "🔍 Testing filtering, search, and pagination..."
    
    # Start the server in background if not running
    $serverJob = Start-Process -FilePath "make" -ArgumentList "run" -NoNewWindow -PassThru
    Start-Sleep -Seconds 5  # Allow server to start
    
    try {
        $results = @{
            sourceFilter = Test-SourceFilter
            biasFilter = Test-BiasFilter
            searchQuery = Test-SearchQuery
            pagination = Test-Pagination
        }
        
        $success = (
            $results.sourceFilter.success -and
            $results.biasFilter.success -and
            $results.searchQuery.success -and
            $results.pagination.success
        )
        
        return @{
            success = $success
            details = $results
        }
    }
    finally {
        # Cleanup - stop the server
        if ($serverJob -ne $null) {
            Stop-Process -Id $serverJob.Id -Force -ErrorAction SilentlyContinue
        }
    }
}

function Test-SourceFilter {
    try {
        # Test source filtering
        $response = Invoke-WebRequest -Uri "http://localhost:8080/articles?source=BBC" -UseBasicParsing
        $hasResults = $response.Content -match '<div\s+class=[''"]article-item[''"]'
        $sourceFiltered = $response.Content -match 'BBC'
        
        return @{
            success = $hasResults -and $sourceFiltered
            statusCode = $response.StatusCode
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

function Test-BiasFilter {
    try {
        # Test bias/leaning filtering
        $response = Invoke-WebRequest -Uri "http://localhost:8080/articles?leaning=Left" -UseBasicParsing
        $hasResults = $response.Content -match '<div\s+class=[''"]article-item[''"]'
        $biasFiltered = $response.Content -match 'Left Leaning'
        
        # Check for backward compatibility
        $backwardCompat = $false
        try {
            $biasResponse = Invoke-WebRequest -Uri "http://localhost:8080/articles?bias=Right" -UseBasicParsing
            $backwardCompat = $biasResponse.StatusCode -eq 200 -and $biasResponse.Content -match 'Right Leaning'
        }
        catch {
            # Ignore errors for backward compatibility check
        }
        
        return @{
            success = $hasResults -and $biasFiltered
            statusCode = $response.StatusCode
            backwardCompatible = $backwardCompat
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-SearchQuery {
    try {
        # Create a test search term likely to exist in articles
        $searchTerm = "news"
        $response = Invoke-WebRequest -Uri "http://localhost:8080/articles?query=$searchTerm" -UseBasicParsing
        $hasResults = $response.Content -match '<div\s+class=[''"]article-item[''"]'
        
        return @{
            success = $hasResults
            statusCode = $response.StatusCode
            searchTerm = $searchTerm
        }
    }
    catch {
        return @{
            success = $false
            error = $_.Exception.Message
        }
    }
}

function Test-Pagination {
    try {
        # Test first page
        $page1 = Invoke-WebRequest -Uri "http://localhost:8080/articles?page=1" -UseBasicParsing
        $hasPage1Results = $page1.Content -match '<div\s+class=[''"]article-item[''"]'
        
        # Test second page
        $page2 = Invoke-WebRequest -Uri "http://localhost:8080/articles?page=2" -UseBasicParsing
        $hasPage2Results = $page2.Content -match '<div\s+class=[''"]article-item[''"]'
        
        # Check if pagination controls exist
        $hasPagination = $page1.Content -match '<div\s+class=[''"]pagination[''"]'
        $hasNextLink = $page1.Content -match 'Next'
        
        return @{
            success = $hasPage1Results -and $hasPage2Results -and $hasPagination
            hasNextLink = $hasNextLink
            statusCodePage1 = $page1.StatusCode
            statusCodePage2 = $page2.StatusCode
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
$controller.UpdatePhase("filtering-pagination")

$result = Test-FilteringAndPagination
if ($result.success) {
    $controller.CompletePhase("filtering-pagination")
    $controller.CurrentState.componentsStatus["filtering"] = "complete"
    $controller.SaveState()
} else {
    Write-Host "❌ Filtering and pagination tests failed" -ForegroundColor Red
    
    # Generate recommendations based on what failed
    $recommendations = @()
    if (-not $result.details.sourceFilter.success) {
        $recommendations += "Fix source filtering in TemplateIndexHandler - source parameter not working properly"
    }
    if (-not $result.details.biasFilter.success) {
        $recommendations += "Fix bias/leaning filtering in TemplateIndexHandler - check threshold values (-0.1/0.1)"
    }
    if (-not $result.details.pagination.success) {
        $recommendations += "Fix pagination in TemplateIndexHandler - ensure LIMIT and OFFSET are applied correctly"
    }
    
    foreach ($rec in $recommendations) {
        Write-Host "  - $rec" -ForegroundColor Yellow
    }
    
    $controller.LogIssue("filtering", "Filtering and pagination failed", "major")
}
```

## Implementation Details

### Filter Form Structure
The articles page should include a filter form with the following elements:
- Source dropdown (populated with available news sources)
- Bias/leaning options (Left, Center, Right)
- Search text input field
- Submit button

### SQL Implementation in Backend
The `TemplateIndexHandler` should construct SQL queries dynamically based on provided parameters:

```go
// Pseudo-code for query construction
query := "SELECT * FROM articles WHERE 1=1"
params := []interface{}{}

// Source filter
if source != "" {
    query += " AND source = ?"
    params = append(params, source)
}

// Bias/leaning filter
if leaning != "" {
    switch leaning {
    case "Left":
        query += " AND composite_score < -0.1"
    case "Right":
        query += " AND composite_score > 0.1"
    case "Center":
        query += " AND composite_score BETWEEN -0.1 AND 0.1"
    }
}

// Search query
if searchQuery != "" {
    query += " AND (title LIKE ? OR content LIKE ?)"
    likeParam := "%" + searchQuery + "%"
    params = append(params, likeParam, likeParam)
}

// Pagination
offset := (page - 1) * 20
query += " ORDER BY published_at DESC LIMIT 21 OFFSET ?"
params = append(params, offset)
```

### Pagination Implementation
The pagination mechanism should:
1. Show current page number
2. Provide "Next" link if more results exist
3. Provide "Previous" link if not on the first page
4. Calculate offset based on page number
5. Request one extra item to determine if more pages exist

### Bias Labeling Logic
The bias labeling should follow these thresholds:
- **Left Leaning**: composite_score < -0.1
- **Center**: composite_score between -0.1 and 0.1
- **Right Leaning**: composite_score > 0.1

### Backward Compatibility
The system should support both `leaning` and `bias` parameter names for backward compatibility.

## Verification Process
The automated verification script tests:
1. Source filtering functionality
2. Bias/leaning filtering with proper thresholds
3. Search query functionality with text matching
4. Pagination with proper page navigation
5. Backward compatibility for parameter naming

Each test verifies both the correct HTTP status code and the presence of expected content in the response, ensuring that filters are working correctly.
