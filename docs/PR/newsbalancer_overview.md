# NewsBalancer Minimal Front‑End Proposal with Autonomous Feedback Loop

## 🔄 Autonomous Feedback Loop Architecture

This implementation incorporates a self-validating, autonomous feedback loop at each stage. The feedback system leverages existing infrastructure and provides real-time verification, ensuring continuous quality throughout development.

### Core Controller Components

```powershell
# feedbackController.ps1 - Main orchestration script that runs throughout implementation
class FeedbackController {
    [string]$WorkspaceRoot = "d:\Dev\newbalancer_go_fe3 - Copy"
    [string]$StateFile = ".\frontend-feedback-state.json"
    [hashtable]$CurrentState = @{}
    
    [void] Initialize() {
        if (Test-Path $this.StateFile) {
            $this.CurrentState = Get-Content $this.StateFile | ConvertFrom-Json -AsHashtable
        } else {
            $this.CurrentState = @{
                startedAt = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
                currentPhase = "environment-verification"
                completedPhases = @()
                currentComponent = $null
                componentsStatus = @{}
                retryCount = 0
                lastSuccess = $null
                issues = @()
            }
        }
        $this.SaveState()
    }
    
    [void] SaveState() {
        $this.CurrentState | ConvertTo-Json -Depth 5 | Set-Content $this.StateFile
    }
    
    [void] UpdatePhase($phaseName) {
        $this.CurrentState.currentPhase = $phaseName
        $this.CurrentState.lastPhaseChange = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        $this.SaveState()
    }
    
    [void] CompletePhase($phaseName) {
        if ($this.CurrentState.currentPhase -eq $phaseName) {
            if (-not $this.CurrentState.completedPhases.Contains($phaseName)) {
                $this.CurrentState.completedPhases += $phaseName
                $this.CurrentState.retryCount = 0
                Write-Host "✅ Phase completed: $phaseName" -ForegroundColor Green
            }
        }
        $this.SaveState()
    }
    
    [void] LogIssue($component, $issue, $severity) {
        $this.CurrentState.issues += @{
            component = $component
            description = $issue
            severity = $severity
            timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            retryCount = $this.CurrentState.retryCount
        }
        $this.CurrentState.retryCount++
        $this.SaveState()
    }
    
    [void] GenerateReport($outputPath) {
        $report = @{
            startedAt = $this.CurrentState.startedAt
            completedAt = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            totalTime = [math]::Round(((Get-Date) - [datetime]::Parse($this.CurrentState.startedAt)).TotalHours, 2)
            completedPhases = $this.CurrentState.completedPhases
            components = $this.CurrentState.componentsStatus
            issues = $this.CurrentState.issues
            summary = @{
                totalPhases = 6
                completedPhases = $this.CurrentState.completedPhases.Count
                successRate = [math]::Round(($this.CurrentState.completedPhases.Count / 6) * 100, 1)
                totalIssues = $this.CurrentState.issues.Count
                criticalIssues = ($this.CurrentState.issues | Where-Object { $_.severity -eq "critical" }).Count
            }
        }
        
        $report | ConvertTo-Json -Depth 5 | Set-Content $outputPath
        Write-Host "📊 Implementation Report generated at: $outputPath" -ForegroundColor Cyan
    }
}
```

## 1️⃣ Overview and Page Routes with Feedback Validation  

The NewsBalancer front-end is a multi-page web interface with server-rendered templates, aligned to specific Go backend handlers. The core pages include: **Articles Listing**, **Article Detail**, and an **Admin Dashboard**. Each page corresponds to a dedicated route and Go template, served by handlers in the backend:

- **Articles List Page** – Accessible at `/articles`, handled by `TemplateIndexHandler` in Go. This route returns the **articles listing page** (template `articles.html`) populated with a list of news articles and their bias summary data.  
- **Article Detail Page** – Accessible at `/article/:id`, handled by `TemplateArticleHandler`. This returns the **article detail page** (`article.html` template) showing the full content of a single article along with its bias analysis results.  
- **Admin Dashboard** – Accessible at `/admin` (handled by `TemplateAdminHandler`), providing summary statistics and controls for maintenance tasks (e.g. feed refresh, metrics).

The front-end uses standard navigation for these routes (full page loads). The backend templates are rendered with real data from the database, ensuring the pages are usable even without client-side JavaScript. For example, the Article Detail template is server-rendered with the article content, its current composite bias score, a bias label ("Left Leaning", "Right Leaning", or "Center"), and a summary of the article (if available). Recent articles and basic stats are also included in sidebars for context.

**Routing Details:** The root path `/` simply redirects to `/articles`. Both the list and detail pages leverage Go templates. The `TemplateIndexHandler` fetches articles from the SQLite DB applying any query filters or pagination before rendering HTML. The `TemplateArticleHandler` fetches the specified article by ID and enriches it with analysis results (composite score, confidence, bias label, summary) prior to template rendering. This alignment ensures that every front-end route has a corresponding backend handler and template, and that any dynamic data displayed has an actual source in the codebase.

## Implementation Phases

This project is structured in 6 distinct phases with integrated validation at each stage:

1. **Core Templates Implementation** - Basic page structures and routes
2. **Filtering, Search, and Pagination** - Query-based data filtering
3. **Article Detail with SSE** - Dynamic analysis and real-time updates
4. **Bias Analysis Data APIs** - JSON endpoints for detailed data access
5. **User Feedback Integration** - User interaction and confidence adjustments
6. **Admin-Only Functions** - Management and monitoring capabilities

Each phase includes autonomous feedback verification to ensure quality and alignment with backend systems.
