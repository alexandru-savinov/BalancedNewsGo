# Test Environment Setup Script
param(
    [switch]$UseContainers,
    [switch]$Verbose,
    [string]$ConfigFile = "configs/test-config.json"
)

$ErrorActionPreference = "Stop"

function Write-EnvLog {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch($Level) {
        "ERROR" { "Red" }
        "WARN" { "Yellow" } 
        "SUCCESS" { "Green" }
        "DEBUG" { "Cyan" }
        default { "White" }
    }
    if ($Verbose -or $Level -ne "DEBUG") {
        Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
    }
}

function Set-TestEnvironment {
    Write-EnvLog "Setting up test environment..." "INFO"
    
    # Set common test environment variables
    $env:NBG_ENV = "test"
    $env:NBG_CONFIG_FILE = $ConfigFile
    $env:NBG_LOG_LEVEL = "debug"
    $env:NBG_DATABASE_TYPE = if ($UseContainers) { "postgres" } else { "sqlite" }
    
    # Database configuration
    if ($UseContainers) {
        Write-EnvLog "Configuring for containerized database testing" "INFO"
        $env:NBG_DATABASE_HOST = "localhost"
        $env:NBG_DATABASE_PORT = "5432"
        $env:NBG_DATABASE_NAME = "testdb"
        $env:NBG_DATABASE_USER = "testuser"
        $env:NBG_DATABASE_PASSWORD = "testpass"
        $env:NBG_DATABASE_SSLMODE = "disable"
    } else {
        Write-EnvLog "Configuring for SQLite testing" "INFO"
        $env:NBG_DATABASE_CONNECTION = ":memory:"
    }
    
    # Mock services configuration
    $env:NBG_LLM_MOCK = "true"
    $env:NBG_RSS_MOCK = "true"
    $env:NBG_METRICS_ENABLED = "false"
    
    # Test-specific settings
    $env:NBG_TEST_TIMEOUT = "60s"
    $env:NBG_TEST_PARALLEL = "true"
    $env:NBG_TEST_CLEANUP = "true"
    
    Write-EnvLog "Test environment configured successfully" "SUCCESS"
}

function Test-Dependencies {
    Write-EnvLog "Checking test dependencies..." "INFO"
    
    $missingDeps = @()
    
    # Check Go
    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        $missingDeps += "go"
    }
    
    # Check Docker if using containers
    if ($UseContainers -and -not (Get-Command "docker" -ErrorAction SilentlyContinue)) {
        $missingDeps += "docker"
    }
    
    # Check Node.js for E2E tests
    if (-not (Get-Command "node" -ErrorAction SilentlyContinue)) {
        Write-EnvLog "Node.js not found - E2E tests will be skipped" "WARN"
    }
    
    if ($missingDeps.Count -gt 0) {
        Write-EnvLog "Missing dependencies: $($missingDeps -join ', ')" "ERROR"
        return $false
    }
    
    Write-EnvLog "All dependencies satisfied" "SUCCESS"
    return $true
}

function Initialize-TestDirectories {
    Write-EnvLog "Initializing test directories..." "INFO"
    
    $testDirs = @(
        "test-results",
        "coverage",
        "testdata",
        "test-results/server-logs",
        "test-results/e2e-results"
    )
    
    foreach ($dir in $testDirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-EnvLog "Created directory: $dir" "DEBUG"
        }
    }
    
    Write-EnvLog "Test directories initialized" "SUCCESS"
}

function Test-Configuration {
    Write-EnvLog "Validating test configuration..." "INFO"
    
    if (-not (Test-Path $ConfigFile)) {
        Write-EnvLog "Configuration file not found: $ConfigFile" "ERROR"
        return $false
    }
    
    try {
        $config = Get-Content $ConfigFile | ConvertFrom-Json
        Write-EnvLog "Configuration file is valid JSON" "DEBUG"
        
        # Validate required sections
        $requiredSections = @("database", "server", "testing")
        foreach ($section in $requiredSections) {
            if (-not $config.$section) {
                Write-EnvLog "Missing required configuration section: $section" "ERROR"
                return $false
            }
        }
        
        Write-EnvLog "Configuration validation passed" "SUCCESS"
        return $true
    }
    catch {
        Write-EnvLog "Invalid configuration file: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Show-EnvironmentInfo {
    Write-EnvLog "Test Environment Information:" "INFO"
    Write-Host "  Config File: $ConfigFile" -ForegroundColor Cyan
    Write-Host "  Database Type: $($env:NBG_DATABASE_TYPE)" -ForegroundColor Cyan
    Write-Host "  Use Containers: $UseContainers" -ForegroundColor Cyan
    Write-Host "  Go Version: $(go version 2>$null)" -ForegroundColor Cyan
    
    if ($UseContainers) {
        $dockerVersion = docker --version 2>$null
        if ($dockerVersion) {
            Write-Host "  Docker Version: $dockerVersion" -ForegroundColor Cyan
        }
    }
}

# Main execution
try {
    Write-EnvLog "Starting test environment setup" "INFO"
    
    # Check dependencies
    if (-not (Test-Dependencies)) {
        exit 1
    }
    
    # Validate configuration
    if (-not (Test-Configuration)) {
        exit 1
    }
    
    # Initialize directories
    Initialize-TestDirectories
    
    # Set environment variables
    Set-TestEnvironment
    
    # Show environment info
    if ($Verbose) {
        Show-EnvironmentInfo
    }
    
    Write-EnvLog "Test environment setup completed successfully" "SUCCESS"
    Write-EnvLog "You can now run tests using the comprehensive test runner" "INFO"
    
    exit 0
}
catch {
    Write-EnvLog "Test environment setup failed: $($_.Exception.Message)" "ERROR"
    exit 1
}
