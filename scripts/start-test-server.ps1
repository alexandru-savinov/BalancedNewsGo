# Test server startup script
param(
    [int]$TimeoutSeconds = 30
)

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
    Write-Host "Starting test server..." -ForegroundColor Yellow
    
    # Check if server is already running
    if (Test-ServerHealth) {
        Write-Host "✅ Server already running and healthy" -ForegroundColor Green
        return $true
    }
      # Start server process
    Write-Host "  Starting Go server process..." -ForegroundColor Blue
    $serverProcess = Start-Process -FilePath "go" -ArgumentList "run", "./cmd/server" -NoNewWindow -PassThru
    
    # Store the process ID for potential cleanup
    $global:TestServerProcessId = $serverProcess.Id
    
    # Wait for server to be ready
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        Start-Sleep -Seconds 2
        $elapsed += 2
        
        if (Test-ServerHealth) {
            Write-Host "✅ Server started successfully in $elapsed seconds" -ForegroundColor Green
            return $true
        }
    }
    
    Write-Host "❌ Server failed to start within $TimeoutSeconds seconds" -ForegroundColor Red
    return $false
}

# Main execution
if (Start-TestServer) {
    Write-Host "✅ Test server is ready for E2E tests" -ForegroundColor Green
    exit 0
} else {
    Write-Host "❌ Failed to start test server" -ForegroundColor Red
    exit 1
}
