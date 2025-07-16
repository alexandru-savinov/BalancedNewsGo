# Comprehensive Test Orchestration Script - Phase 2 Enhanced with Testcontainers
param(
    [ValidateSet("all", "unit", "integration", "e2e", "security", "load")]
    [string]$TestType = "all",
    [switch]$SkipServerStart,
    [switch]$Verbose,
    [switch]$Coverage,
    [switch]$UsePostgres,
    [switch]$UseSQLite,
    [switch]$Parallel,
    [int]$TimeoutMinutes = 10
)

$ErrorActionPreference = "Stop"
$ResultsDir = "test-results"
$CoverageDir = "coverage"

# Enhanced test configuration with testcontainers support
$TestConfig = @{
    UnitTestTimeout = 120
    IntegrationTestTimeout = 300
    E2ETestTimeout = 600
    PerformanceTestTimeout = 900
    ServerStartupTimeout = 30
    DatabaseSetupTimeout = 120
    ContainerPullTimeout = 300
    TestContainersEnabled = $true
    DefaultDatabaseType = if ($UsePostgres) { "postgres" } elseif ($UseSQLite) { "sqlite" } else { "postgres" }
}

# Logging with timestamps and colors
function Write-TestLog {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch($Level) {
        "ERROR" { "Red" }
        "WARN" { "Yellow" } 
        "SUCCESS" { "Green" }
        "DEBUG" { "Cyan" }
        default { "White" }
    }
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

# Enhanced server health check with retry logic
function Test-ServerHealth {
    param([int]$MaxRetries = 5, [int]$DelaySeconds = 2)
    
    for ($i = 1; $i -le $MaxRetries; $i++) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/healthz" -TimeoutSec 5 -Method Get
            if ($response.status -eq "ok" -or $response -eq "OK") {
                Write-TestLog "Server health check passed (attempt $i/$MaxRetries)" "SUCCESS"
                return $true
            }
        }
        catch {
            Write-TestLog "Health check attempt $i/$MaxRetries failed: $($_.Exception.Message)" "DEBUG"
        }
        
        if ($i -lt $MaxRetries) {
            Start-Sleep -Seconds $DelaySeconds
        }
    }
    
    Write-TestLog "Server health check failed after $MaxRetries attempts" "ERROR"
    return $false
}

# Advanced server startup with process management
function Start-TestServer {
    if ($SkipServerStart) {
        Write-TestLog "Skipping server startup as requested" "INFO"
        return $null
    }
    
    Write-TestLog "Checking existing server..." "INFO"
    if (Test-ServerHealth -MaxRetries 1) {
        Write-TestLog "Server already running and healthy" "SUCCESS"
        return $null
    }
    
    # Kill any existing server processes
    Get-Process | Where-Object { $_.ProcessName -eq "go" -and $_.CommandLine -like "*cmd/server*" } | ForEach-Object {
        Write-TestLog "Terminating existing server process (PID: $($_.Id))" "WARN"
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
    
    Write-TestLog "Starting new server process..." "INFO"
    
    # Create server logs directory
    $serverLogsDir = "$ResultsDir/server-logs"
    New-Item -ItemType Directory -Path $serverLogsDir -Force | Out-Null
    
    try {
        # Start server with output redirection
        $processStartInfo = New-Object System.Diagnostics.ProcessStartInfo
        $processStartInfo.FileName = "go"
        $processStartInfo.Arguments = "run ./cmd/server"
        $processStartInfo.UseShellExecute = $false
        $processStartInfo.RedirectStandardOutput = $true
        $processStartInfo.RedirectStandardError = $true
        $processStartInfo.CreateNoWindow = $true
        $processStartInfo.WorkingDirectory = Get-Location
        
        $serverProcess = [System.Diagnostics.Process]::Start($processStartInfo)
        
        # Wait for server to be ready
        $timeout = (Get-Date).AddMinutes($TimeoutMinutes)
        $serverReady = $false
        
        while ((Get-Date) -lt $timeout -and !$serverReady) {
            Start-Sleep -Seconds 2
            $serverReady = Test-ServerHealth -MaxRetries 1
            
            if ($serverProcess.HasExited) {
                throw "Server process exited unexpectedly"
            }
        }
        
        if ($serverReady) {
            Write-TestLog "Server started successfully (PID: $($serverProcess.Id))" "SUCCESS"
            return $serverProcess
        } else {
            throw "Server failed to start within $TimeoutMinutes minutes"
        }
    }
    catch {
        Write-TestLog "Server startup failed: $($_.Exception.Message)" "ERROR"
        throw
    }
}

# Testcontainer management functions
function Test-DockerAvailability {
    try {
        $dockerVersion = docker version --format '{{.Server.Version}}' 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-TestLog "Docker is available (version: $dockerVersion)" "SUCCESS"
            return $true
        }
    }
    catch {
        Write-TestLog "Docker check failed: $($_.Exception.Message)" "ERROR"
    }
    
    Write-TestLog "Docker is not available - testcontainers will not work" "ERROR"
    return $false
}

function Initialize-TestContainers {
    Write-TestLog "Initializing testcontainers environment..." "INFO"
    
    if (-not (Test-DockerAvailability)) {
        Write-TestLog "Skipping testcontainer initialization - Docker not available" "WARN"
        return $false
    }
    
    # Pull required container images
    $requiredImages = @(
        "postgres:15-alpine",
        "testcontainers/ryuk:0.5.1"
    )
    
    foreach ($image in $requiredImages) {
        Write-TestLog "Pulling container image: $image" "INFO"
        try {
            docker pull $image | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-TestLog "Successfully pulled $image" "SUCCESS"
            } else {
                Write-TestLog "Failed to pull $image" "WARN"
            }
        }
        catch {
            Write-TestLog "Error pulling ${image}: $($_.Exception.Message)" "WARN"
        }
    }
    
    # Set testcontainers environment variables
    $env:TESTCONTAINERS_RYUK_DISABLED = "false"
    $env:TESTCONTAINERS_CHECKS_DISABLE = "false"
    
    Write-TestLog "Testcontainers environment initialized" "SUCCESS"
    return $true
}

function Clear-TestContainers {
    Write-TestLog "Cleaning up testcontainers..." "INFO"
    
    try {
        # Remove testcontainer networks
        docker network ls --format "{{.Name}}" | Where-Object { $_ -like "*testcontainers*" } | ForEach-Object {
            Write-TestLog "Removing testcontainer network: $_" "DEBUG"
            docker network rm $_ 2>$null | Out-Null
        }
        
        # Remove testcontainer volumes
        docker volume ls --format "{{.Name}}" | Where-Object { $_ -like "*testcontainers*" } | ForEach-Object {
            Write-TestLog "Removing testcontainer volume: $_" "DEBUG"
            docker volume rm $_ 2>$null | Out-Null
        }
        
        Write-TestLog "Testcontainer cleanup completed" "SUCCESS"
    }
    catch {
        Write-TestLog "Error during testcontainer cleanup: $($_.Exception.Message)" "WARN"
    }
}

# Enhanced test execution functions
function Invoke-UnitTestsWithContainers {
    Write-TestLog "Running unit tests with testcontainer support..." "INFO"
    
    $testCommand = "go test"
    $testArgs = @(
        "./internal/...",
        "-v"
    )
    
    if ($Coverage) {
        $testArgs += "-coverprofile=$CoverageDir/unit-coverage.out"
        $testArgs += "-covermode=atomic"
    }
    
    if ($Parallel) {
        $testArgs += "-parallel=4"
    }
    
    $testArgs += "-timeout=$($TestConfig.UnitTestTimeout)s"
      # Set environment variables for testcontainers
    $env:GO_TEST_TIMEOUT = "$($TestConfig.UnitTestTimeout)s"
    $env:TESTCONTAINERS_DATABASE_TYPE = $TestConfig.DefaultDatabaseType
    
    & $testCommand @testArgs | Tee-Object -FilePath "$ResultsDir/unit-test-output.log"
    $exitCode = $LASTEXITCODE
    
    if ($exitCode -eq 0) {
        Write-TestLog "Unit tests passed successfully" "SUCCESS"
    } else {
        Write-TestLog "Unit tests failed with exit code: $exitCode" "ERROR"
    }
    
    return $exitCode
}

function Invoke-IntegrationTestsWithContainers {
    Write-TestLog "Running integration tests with testcontainers..." "INFO"
    
    $testCommand = "go test"
    $testArgs = @(
        "./tests/...",
        "-v",
        "-tags=integration"
    )
    
    if ($Coverage) {
        $testArgs += "-coverprofile=$CoverageDir/integration-coverage.out"
        $testArgs += "-covermode=atomic"
    }
    
    $testArgs += "-timeout=$($TestConfig.IntegrationTestTimeout)s"
      # Set environment variables for testcontainers
    $env:GO_TEST_TIMEOUT = "$($TestConfig.IntegrationTestTimeout)s"
    $env:TESTCONTAINERS_DATABASE_TYPE = $TestConfig.DefaultDatabaseType
    $env:TESTCONTAINERS_STARTUP_TIMEOUT = "$($TestConfig.DatabaseSetupTimeout)s"
    
    & $testCommand @testArgs | Tee-Object -FilePath "$ResultsDir/integration-test-output.log"
    $exitCode = $LASTEXITCODE
    
    if ($exitCode -eq 0) {
        Write-TestLog "Integration tests passed successfully" "SUCCESS"
    } else {
        Write-TestLog "Integration tests failed with exit code: $exitCode" "ERROR"
    }
    
    return $exitCode
}

# Test execution functions
function Invoke-UnitTests {
    Write-TestLog "Running Go unit tests..." "INFO"
    
    $testArgs = @("test", "./...")
    
    if ($Verbose) {
        $testArgs += "-v"
    }
    
    if ($Coverage) {
        New-Item -ItemType Directory -Path $CoverageDir -Force | Out-Null
        $testArgs += "-coverprofile=$CoverageDir/unit-coverage.out"
        $testArgs += "-covermode=atomic"
    }
    
    $testArgs += "-json"
    
    & go @testArgs | Tee-Object -FilePath "$ResultsDir/unit-tests.json"
    return $LASTEXITCODE
}

function Invoke-IntegrationTests {
    Write-TestLog "Running integration tests..." "INFO"
    
    # Example integration test command - adjust based on your setup
    $testCommand = @("test", "-tags=integration", "./tests/integration/...")
    
    if ($Verbose) {
        $testCommand += "-v"
    }
    
    & go @testCommand | Tee-Object -FilePath "$ResultsDir/integration-tests.log"
    return $LASTEXITCODE
}

function Invoke-E2ETests {
    Write-TestLog "Running optimized E2E tests..." "INFO"

    # Ensure Playwright dependencies are available
    if (Get-Command "npx" -ErrorAction SilentlyContinue) {
        npx playwright test tests/e2e/ --reporter=dot --timeout=60000 --output-dir="$ResultsDir/e2e-results" | Tee-Object -FilePath "$ResultsDir/e2e-tests.log"
        return $LASTEXITCODE
    } else {
        Write-TestLog "npx not found - skipping E2E tests" "WARN"
        return 0
    }
}

function Invoke-SecurityTests {
    Write-TestLog "Running security tests..." "INFO"
    
    if (Get-Command "gosec" -ErrorAction SilentlyContinue) {
        gosec -fmt json -out "$ResultsDir/security-report.json" ./...
        $securityExitCode = $LASTEXITCODE
        
        # Also run with human-readable output
        gosec ./... | Tee-Object -FilePath "$ResultsDir/security-tests.log"
        
        return $securityExitCode
    } else {
        Write-TestLog "gosec not installed - skipping security tests" "WARN"
        return 0
    }
}

# Main execution logic
try {
    Write-TestLog "Starting comprehensive test execution: $TestType" "INFO"
    
    # Initialize results directory
    if (Test-Path $ResultsDir) {
        Remove-Item $ResultsDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $ResultsDir -Force | Out-Null
    
    # Start server if needed
    $serverProcess = Start-TestServer
    
    # Execute tests based on type
    $exitCodes = @{}
    
    switch ($TestType.ToLower()) {
        "unit" {
            $exitCodes["unit"] = Invoke-UnitTests
        }
        "integration" {
            $exitCodes["integration"] = Invoke-IntegrationTests
        }
        "e2e" {
            $exitCodes["e2e"] = Invoke-E2ETests
        }
        "security" {
            $exitCodes["security"] = Invoke-SecurityTests
        }
        "all" {
            $exitCodes["unit"] = Invoke-UnitTests
            $exitCodes["integration"] = Invoke-IntegrationTests
            $exitCodes["e2e"] = Invoke-E2ETests
            $exitCodes["security"] = Invoke-SecurityTests
        }
    }
    
    # Generate test summary
    $summary = @{
        Timestamp = Get-Date
        TestType = $TestType
        Results = $exitCodes
        ServerPID = if ($serverProcess) { $serverProcess.Id } else { $null }
    }
    
    $summary | ConvertTo-Json -Depth 3 | Set-Content "$ResultsDir/test-summary.json"
    
    # Determine overall success
    $failures = $exitCodes.Values | Where-Object { $_ -ne 0 }
    
    if ($failures.Count -eq 0) {
        Write-TestLog "All tests completed successfully!" "SUCCESS"
        $overallExitCode = 0
    } else {
        Write-TestLog "Some tests failed. Check individual results for details." "ERROR"
        $overallExitCode = 1
    }
    
    # Coverage report if enabled
    if ($Coverage -and (Test-Path "$CoverageDir/unit-coverage.out")) {
        Write-TestLog "Generating coverage report..." "INFO"
        go tool cover -html="$CoverageDir/unit-coverage.out" -o "$CoverageDir/coverage.html"
        go tool cover -func="$CoverageDir/unit-coverage.out" | Tee-Object -FilePath "$CoverageDir/coverage-summary.txt"
    }
    
    exit $overallExitCode
}
catch {
    Write-TestLog "Test execution failed: $($_.Exception.Message)" "ERROR"
    exit 1
}
finally {
    # Cleanup server process
    if ($serverProcess -and !$serverProcess.HasExited) {
        Write-TestLog "Cleaning up server process..." "INFO"
        Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    }
}
