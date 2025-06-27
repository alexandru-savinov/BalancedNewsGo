# E2E SSE Monitoring Script
# Monitors SSE stream for reanalysis completion

param(
    [int]$ArticleId = 587,
    [int]$TimeoutSeconds = 60,
    [int]$PollIntervalSeconds = 2
)

$startTime = Get-Date
$endTime = $startTime.AddSeconds($TimeoutSeconds)

Write-Host "[$((Get-Date).ToString('HH:mm:ss'))] Starting SSE monitoring for article $ArticleId" -ForegroundColor Green
Write-Host "[$((Get-Date).ToString('HH:mm:ss'))] Timeout: $TimeoutSeconds seconds" -ForegroundColor Gray
Write-Host "[$((Get-Date).ToString('HH:mm:ss'))] Poll interval: $PollIntervalSeconds seconds" -ForegroundColor Gray
Write-Host ""

$lastStatus = $null
$progressHistory = @()

while ((Get-Date) -lt $endTime) {
    try {
        $timestamp = (Get-Date).ToString('HH:mm:ss')
        
        # Poll the SSE endpoint
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/llm/score-progress/$ArticleId" -Headers @{"Accept"="text/event-stream"} -UseBasicParsing -TimeoutSec 5
        
        if ($response.StatusCode -eq 200) {
            # Parse SSE data
            $content = $response.Content
            if ($content -match 'data: ({.*})') {
                $jsonData = $matches[1]
                try {
                    $data = $jsonData | ConvertFrom-Json
                    
                    # Track progress
                    $progressHistory += [PSCustomObject]@{
                        Timestamp = Get-Date
                        Status = $data.status
                        Step = $data.step
                        Message = $data.message
                        Progress = if ($data.percent) { $data.percent } else { $data.progress }
                        Score = $data.final_score
                    }
                    
                    # Only log status changes
                    if ($data.status -ne $lastStatus) {
                        $lastStatus = $data.status
                        Write-Host "[$timestamp] Status: $($data.status)" -ForegroundColor Cyan
                        if ($data.step) { Write-Host "[$timestamp] Step: $($data.step)" -ForegroundColor White }
                        if ($data.message) { Write-Host "[$timestamp] Message: $($data.message)" -ForegroundColor Gray }
                        if ($data.percent -or $data.progress) { 
                            $prog = if ($data.percent) { $data.percent } else { $data.progress }
                            Write-Host "[$timestamp] Progress: $prog%" -ForegroundColor Yellow 
                        }
                    }
                    
                    # Check for completion
                    if ($data.status -eq "Complete" -or $data.status -eq "completed" -or $data.status -eq "Success") {
                        Write-Host ""
                        Write-Host "[$timestamp] ✅ ANALYSIS COMPLETED!" -ForegroundColor Green
                        Write-Host "[$timestamp] Final Status: $($data.status)" -ForegroundColor Green
                        if ($data.final_score) {
                            Write-Host "[$timestamp] Final Score: $($data.final_score)" -ForegroundColor Green
                        }
                        
                        # Return completion data
                        return [PSCustomObject]@{
                            Success = $true
                            FinalStatus = $data.status
                            FinalScore = $data.final_score
                            Duration = ((Get-Date) - $startTime).TotalSeconds
                            ProgressHistory = $progressHistory
                        }
                    }
                    
                    # Check for error states
                    if ($data.status -eq "Error" -or $data.status -eq "Failed") {
                        Write-Host ""
                        Write-Host "[$timestamp] ❌ ANALYSIS FAILED!" -ForegroundColor Red
                        Write-Host "[$timestamp] Error Status: $($data.status)" -ForegroundColor Red
                        if ($data.message) {
                            Write-Host "[$timestamp] Error Message: $($data.message)" -ForegroundColor Red
                        }
                        
                        return [PSCustomObject]@{
                            Success = $false
                            Error = "Analysis failed with status: $($data.status)"
                            ErrorMessage = $data.message
                            Duration = ((Get-Date) - $startTime).TotalSeconds
                            ProgressHistory = $progressHistory
                        }
                    }
                    
                } catch {
                    Write-Host "[$timestamp] ⚠️ Failed to parse JSON: $jsonData" -ForegroundColor Yellow
                }
            } else {
                Write-Host "[$timestamp] ⚠️ No JSON data found in SSE response" -ForegroundColor Yellow
            }
        } else {
            Write-Host "[$timestamp] ⚠️ HTTP $($response.StatusCode): $($response.StatusDescription)" -ForegroundColor Yellow
        }
        
    } catch {
        Write-Host "[$timestamp] ❌ Request failed: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    # Wait before next poll
    Start-Sleep -Seconds $PollIntervalSeconds
}

# Timeout reached
Write-Host ""
Write-Host "[$((Get-Date).ToString('HH:mm:ss'))] ⏰ TIMEOUT REACHED!" -ForegroundColor Red
Write-Host "Analysis did not complete within $TimeoutSeconds seconds" -ForegroundColor Red

return [PSCustomObject]@{
    Success = $false
    Error = "Timeout after $TimeoutSeconds seconds"
    Duration = $TimeoutSeconds
    ProgressHistory = $progressHistory
}
