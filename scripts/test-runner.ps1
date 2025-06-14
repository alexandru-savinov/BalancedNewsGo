# Test Runner Script for NewBalancer Go
# Usage: .\scripts\test-runner.ps1 -TestType "all" -Verbose

param(
    [string]$TestType = "all",
    [switch]$SkipServerStart,
    [switch]$Verbose,
    [switch]$StopOnFailure
)

$ErrorActionPreference = "Stop"
$ResultsDir = "test-results"

function Write-Status {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $(
        switch($Level) {
            "ERROR" { "Red" }
            "WARN" { "Yellow" }
            "SUCCESS" { "Green" }
            "INFO" { "Cyan" }
            default { "White" }
        }
    )
}

function Test-ServerHealth {
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/healthz" -TimeoutSec 5
        return $response.status -eq "ok"
    }
    catch {
        return $false
    }
}

function Start-TestServer {
    if (-not $SkipServerStart) {
        Write-Status "Checking if server is running..."
        if (Test-ServerHealth) {
            Write-Status "Server already running" "SUCCESS"
            return $null
        }
        
        Write-Status "Starting Go server..."
        $serverProcess = Start-Process -FilePath "go" -ArgumentList "run", "./cmd/server" -NoNewWindow -PassThru
        
        # Wait for server to start
        $timeout = 30
        $elapsed = 0
        while ($elapsed -lt $timeout) {
            Start-Sleep -Seconds 1
            $elapsed++
            if (Test-ServerHealth) {
                Write-Status "Server started successfully (PID: $($serverProcess.Id))" "SUCCESS"
                return $serverProcess
            }
            if ($Verbose) {
                Write-Status "Waiting for server... ($elapsed/$timeout)" "INFO"
            }
        }
        
        Write-Status "Server failed to start within $timeout seconds" "ERROR"
        if ($serverProcess -and -not $serverProcess.HasExited) {
            $serverProcess.Kill()
        }
        throw "Server startup failed"
    }
    return $null
}

function Stop-TestServer {
    param($Process)
    if ($Process -and -not $Process.HasExited) {
        Write-Status "Stopping test server (PID: $($Process.Id))..."
        try {
            $Process.Kill()
            $Process.WaitForExit(5000)
            Write-Status "Server stopped" "SUCCESS"
        }
        catch {
            Write-Status "Error stopping server: $($_.Exception.Message)" "WARN"
        }
    }
}

function Run-GoTests {
    Write-Status "Running Go unit tests..."
    $logFile = "$ResultsDir/go-test-results-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
    
    # First check if code compiles
    Write-Status "Verifying Go compilation..."
    $buildOutput = & go build ./... 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Status "Go compilation failed!" "ERROR"
        $buildOutput | Out-File -FilePath $logFile
        Write-Status "Build errors saved to: $logFile" "ERROR"
        return $false
    }
    
    # Run tests with verbose output
    $testOutput = & go test -v ./... 2>&1
    $testOutput | Tee-Object -FilePath $logFile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Status "Go tests completed successfully" "SUCCESS"
        return $true
    } else {
        Write-Status "Go tests failed with exit code: $LASTEXITCODE" "ERROR"
        return $false
    }
}

function Run-E2ETests {
    Write-Status "Running E2E tests..."
    $logFile = "$ResultsDir/e2e-test-results-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
    
    # Verify server is accessible
    if (-not (Test-ServerHealth)) {
        Write-Status "Server health check failed before E2E tests" "ERROR"
        return $false
    }
    
    $testArgs = @("test")
    if ($Verbose) {
        $testArgs += "--reporter=list"
    }
    
    $testOutput = & npx playwright @testArgs 2>&1
    $testOutput | Tee-Object -FilePath $logFile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Status "E2E tests completed successfully" "SUCCESS"
        return $true
    } else {
        Write-Status "E2E tests failed with exit code: $LASTEXITCODE" "ERROR"
        return $false
    }
}

function Run-IntegrationTests {
    Write-Status "Running integration tests..."
    $logFile = "$ResultsDir/integration-test-results-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
    
    # Run Newman-based API tests
    $testOutput = & npm run test:backend 2>&1
    $testOutput | Tee-Object -FilePath $logFile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Status "Integration tests completed successfully" "SUCCESS"
        return $true
    } else {
        Write-Status "Integration tests failed with exit code: $LASTEXITCODE" "ERROR"
        return $false
    }
}

function Generate-TestReport {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $reportFile = "$ResultsDir/test-summary-$(Get-Date -Format 'yyyyMMdd-HHmmss').html"
    
    $html = @"
<!DOCTYPE html>
<html>
<head>
    <title>Test Results Summary - $timestamp</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .warn { color: orange; }
        .test-section { margin: 20px 0; padding: 15px; border-left: 4px solid #ccc; }
        .test-section.success { border-left-color: green; }
        .test-section.error { border-left-color: red; }
        .log-link { margin-left: 10px; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="header">
        <h1>NewBalancer Go - Test Results</h1>
        <p><strong>Generated:</strong> $timestamp</p>
        <p><strong>Test Type:</strong> $TestType</p>
        <p><strong>Environment:</strong> Development</p>
    </div>
"@

    # Add test results
    if ($script:GoTestsResult -ne $null) {
        $status = if ($script:GoTestsResult) { "success" } else { "error" }
        $statusText = if ($script:GoTestsResult) { "PASSED" } else { "FAILED" }
        $html += "<div class='test-section $status'><h3>Go Unit Tests: $statusText</h3></div>"
    }
    
    if ($script:E2ETestsResult -ne $null) {
        $status = if ($script:E2ETestsResult) { "success" } else { "error" }
        $statusText = if ($script:E2ETestsResult) { "PASSED" } else { "FAILED" }
        $html += "<div class='test-section $status'><h3>E2E Tests: $statusText</h3></div>"
    }
    
    if ($script:IntegrationTestsResult -ne $null) {
        $status = if ($script:IntegrationTestsResult) { "success" } else { "error" }
        $statusText = if ($script:IntegrationTestsResult) { "PASSED" } else { "FAILED" }
        $html += "<div class='test-section $status'><h3>Integration Tests: $statusText</h3></div>"
    }
    
    $html += @"
    <div class="test-section">
        <h3>Log Files</h3>
        <ul>
"@
    
    Get-ChildItem -Path $ResultsDir -Filter "*-$(Get-Date -Format 'yyyyMMdd')*" | ForEach-Object {
        $html += "<li><a href='$($_.Name)'>$($_.Name)</a></li>"
    }
    
    $html += @"
        </ul>
    </div>
</body>
</html>
"@
    
    Set-Content -Path $reportFile -Value $html
    Write-Status "Test report generated: $reportFile" "SUCCESS"
}

# Main execution
try {
    Write-Status "🚀 Starting test execution for: $TestType"
    Write-Status "Working directory: $(Get-Location)"
    
    # Ensure results directory exists
    if (-not (Test-Path $ResultsDir)) {
        New-Item -ItemType Directory -Path $ResultsDir | Out-Null
        Write-Status "Created results directory: $ResultsDir"
    }
    
    # Start server if needed
    $serverProcess = Start-TestServer
    
    # Initialize result tracking
    $script:GoTestsResult = $null
    $script:E2ETestsResult = $null
    $script:IntegrationTestsResult = $null
    $overallSuccess = $true
    
    # Run tests based on type
    switch ($TestType.ToLower()) {
        "go" {
            $script:GoTestsResult = Run-GoTests
            $overallSuccess = $script:GoTestsResult
        }
        "e2e" {
            $script:E2ETestsResult = Run-E2ETests
            $overallSuccess = $script:E2ETestsResult
        }
        "integration" {
            $script:IntegrationTestsResult = Run-IntegrationTests
            $overallSuccess = $script:IntegrationTestsResult
        }
        "all" {
            Write-Status "Running comprehensive test suite..."
            
            # Run Go tests first
            $script:GoTestsResult = Run-GoTests
            if (-not $script:GoTestsResult -and $StopOnFailure) {
                throw "Go tests failed and StopOnFailure is enabled"
            }
            
            # Run E2E tests
            $script:E2ETestsResult = Run-E2ETests
            if (-not $script:E2ETestsResult -and $StopOnFailure) {
                throw "E2E tests failed and StopOnFailure is enabled"
            }
            
            # Run Integration tests
            $script:IntegrationTestsResult = Run-IntegrationTests
            if (-not $script:IntegrationTestsResult -and $StopOnFailure) {
                throw "Integration tests failed and StopOnFailure is enabled"
            }
            
            $overallSuccess = $script:GoTestsResult -and $script:E2ETestsResult -and $script:IntegrationTestsResult
        }
        default {
            throw "Unknown test type: $TestType. Valid options: go, e2e, integration, all"
        }
    }
    
    # Generate test report
    Generate-TestReport
    
    # Final status
    if ($overallSuccess) {
        Write-Status "🎉 All tests completed successfully!" "SUCCESS"
        exit 0
    } else {
        Write-Status "⚠️ Some tests failed. Check logs for details." "WARN"
        exit 1
    }
}
catch {
    Write-Status "💥 Test execution failed: $($_.Exception.Message)" "ERROR"
    if ($Verbose) {
        Write-Status "Stack trace: $($_.ScriptStackTrace)" "ERROR"
    }
    exit 1
}
finally {
    # Cleanup
    if ($serverProcess) {
        Stop-TestServer $serverProcess
    }
    Write-Status "Test execution completed at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
}
