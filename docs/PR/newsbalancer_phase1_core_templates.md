# NewsBalancer Phase 1: Core Templates Implementation

## Core Templates and Basic Structure

The NewsBalancer front-end is a multi-page web interface with server-rendered templates, aligned to specific Go backend handlers. The core pages include:

- **Articles List Page** – Accessible at `/articles`, handled by `TemplateIndexHandler` in Go. This route returns the **articles listing page** (template `articles.html`) populated with a list of news articles and their bias summary data.  
- **Article Detail Page** – Accessible at `/article/:id`, handled by `TemplateArticleHandler`. This returns the **article detail page** (`article.html` template) showing the full content of a single article along with its bias analysis results.  
- **Admin Dashboard** – Accessible at `/admin` (handled by `TemplateAdminHandler`), providing summary statistics and controls for maintenance tasks (e.g. feed refresh, metrics).

The front-end uses standard navigation for these routes (full page loads). The backend templates are rendered with real data from the database, ensuring the pages are usable even without client-side JavaScript. For example, the Article Detail template is server-rendered with the article content, its current composite bias score, a bias label ("Left Leaning", "Right Leaning", or "Center"), and a summary of the article (if available). Recent articles and basic stats are also included in sidebars for context.

**Routing Details:** The root path `/` simply redirects to `/articles`. Both the list and detail pages leverage Go templates. The `TemplateIndexHandler` fetches articles from the SQLite DB applying any query filters or pagination before rendering HTML. The `TemplateArticleHandler` fetches the specified article by ID and enriches it with analysis results (composite score, confidence, bias label, summary) prior to template rendering. This alignment ensures that every front-end route has a corresponding backend handler and template, and that any dynamic data displayed has an actual source in the codebase.

## 🔄 Core Templates Implementation & Verification

```powershell
# phase1_templates.ps1 - Executed after initial template implementation
function Verify-CoreTemplates {
    Write-Host "🔍 Verifying core templates implementation..."
    
    $results = @{
        templates = @{}
        routes = @{}
        overall = $false
    }
    
    # 1. Verify template files exist
    $templatePaths = @{
        articles = Join-Path $controller.WorkspaceRoot "templates\articles.html"
        article = Join-Path $controller.WorkspaceRoot "templates\article.html"
        admin = Join-Path $controller.WorkspaceRoot "templates\admin.html"
    }
    
    foreach ($template in $templatePaths.Keys) {
        $exists = Test-Path $templatePaths[$template]
        $results.templates[$template] = @{ exists = $exists }
        
        if ($exists) {
            # Check template content for expected elements
            $content = Get-Content $templatePaths[$template] -Raw
            
            switch ($template) {
                "articles" {
                    $results.templates[$template].hasArticleList = $content -match '<div\s+class=[''"]article-list[''"]'
                    $results.templates[$template].hasFilters = $content -match '<form\s+class=[''"]filter-form[''"]'
                    $results.templates[$template].hasPagination = $content -match '<div\s+class=[''"]pagination[''"]'
                }
                "article" {
                    $results.templates[$template].hasContent = $content -match '<div\s+class=[''"]article-content[''"]'
                    $results.templates[$template].hasBiasAnalysis = $content -match '<div\s+class=[''"]bias-analysis[''"]'
                    $results.templates[$template].hasReanalysisButton = $content -match '<button[^>]*data-action=[''"]reanalyze[''"]'
                }
                "admin" {
                    $results.templates[$template].hasDashboard = $content -match '<div\s+class=[''"]admin-dashboard[''"]'
                    $results.templates[$template].hasControls = $content -match '<div\s+class=[''"]admin-controls[''"]'
                }
            }
        }
    }
    
    # 2. Verify routes & handlers
    try {
        $serverFiles = Get-ChildItem -Path (Join-Path $controller.WorkspaceRoot "cmd\server") -Filter "*.go" -Recurse
        $serverContent = $serverFiles | Get-Content -Raw
        $serverContentStr = $serverContent -join "`n"
        
        $results.routes.indexHandler = $serverContentStr -match "TemplateIndexHandler"
        $results.routes.articleHandler = $serverContentStr -match "TemplateArticleHandler"
        $results.routes.adminHandler = $serverContentStr -match "TemplateAdminHandler"
        $results.routes.rootRedirect = $serverContentStr -match '"".*"/articles"'
    }
    catch {
        $results.routes.error = $_.Exception.Message
    }
    
    # 3. Overall validation
    $results.overall = (
        $results.templates.articles.exists -and
        $results.templates.article.exists -and
        $results.templates.admin.exists -and
        $results.routes.indexHandler -and
        $results.routes.articleHandler -and
        $results.routes.adminHandler
    )
    
    # 4. Generate recommendations for any issues
    $recommendations = @()
    if (-not $results.templates.articles.exists) {
        $recommendations += "Create the articles.html template in the templates directory"
    }
    elseif (-not $results.templates.articles.hasArticleList) {
        $recommendations += "Add an article-list container div to articles.html"
    }
    
    if (-not $results.templates.article.exists) {
        $recommendations += "Create the article.html template in the templates directory"
    }
    elseif (-not $results.templates.article.hasReanalysisButton) {
        $recommendations += "Add a reanalysis button with data-action='reanalyze' to article.html"
    }
    
    if (-not $results.routes.indexHandler) {
        $recommendations += "Implement the TemplateIndexHandler for /articles route"
    }
    
    return @{
        success = $results.overall
        details = $results
        recommendations = $recommendations
    }
}

# Integration with feedback controller
$controller = [FeedbackController]::new()
$controller.Initialize()
$controller.UpdatePhase("core-templates")

$result = Verify-CoreTemplates
if ($result.success) {
    $controller.CompletePhase("core-templates")
} else {
    Write-Host "❌ Core templates verification failed" -ForegroundColor Red
    foreach ($rec in $result.recommendations) {
        Write-Host "  - $rec" -ForegroundColor Yellow
    }
    $controller.LogIssue("templates", "Core templates incomplete", "major")
}
```

## Implementation Requirements

Each template should contain the following key elements:

### Articles List Template (`articles.html`)
- Article list container (`<div class="article-list">`)
- Filter form (`<form class="filter-form">`)
- Pagination controls (`<div class="pagination">`)
- Each article entry should display:
  - Title
  - Source
  - Publication date
  - Bias indicator showing the composite score

### Article Detail Template (`article.html`)
- Article content section (`<div class="article-content">`)
- Bias analysis display (`<div class="bias-analysis">`)
- Reanalysis button with `data-action="reanalyze"` attribute
- Article metadata (source, date, author)
- Bias score visualization
- Article summary (if available)

### Admin Dashboard Template (`admin.html`)
- Admin dashboard container (`<div class="admin-dashboard">`)
- Admin controls section (`<div class="admin-controls">`)
- Statistics and metrics visualization
- Maintenance controls

## Verification Process

The automated verification script checks:
1. Existence of all template files
2. Presence of required HTML elements in each template
3. Implementation of required Go handlers in the server code
4. Root path redirection to `/articles`

Upon successful verification, the phase is marked as complete. If issues are detected, specific recommendations are provided to address them.
