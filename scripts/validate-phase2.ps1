# Phase 2 Implementation Validation Script - Enhanced with Testcontainer Support
param(
    [switch]$Verbose,
    [switch]$SkipContainerTests,
    [switch]$FixIssues
)

$ErrorActionPreference = "Stop"

function Write-ValidationLog {
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

function Test-FileExists {
    param([string]$FilePath, [string]$Description)
    
    if (Test-Path $FilePath) {
        Write-ValidationLog "✅ $Description exists: $FilePath" "SUCCESS"
        return $true
    } else {
        Write-ValidationLog "❌ $Description missing: $FilePath" "ERROR"
        return $false
    }
}

function Test-GoModules {
    Write-ValidationLog "Checking Go module dependencies..." "INFO"
    
    $requiredModules = @(
        "github.com/testcontainers/testcontainers-go",
        "github.com/testcontainers/testcontainers-go/modules/postgres",
        "github.com/mattn/go-sqlite3",
        "github.com/stretchr/testify"
    )
    
    $goModContent = Get-Content "go.mod" -Raw -ErrorAction SilentlyContinue
    $allFound = $true
    
    foreach ($module in $requiredModules) {
        if ($goModContent -and $goModContent.Contains($module)) {
            Write-ValidationLog "✅ Required module found: $module" "SUCCESS"
        } else {
            Write-ValidationLog "⚠️  Required module not found in go.mod: $module" "WARN"
            Write-ValidationLog "   Run: go get $module" "INFO"
            $allFound = $false
        }
    }
    
    return $allFound
}

function Test-ScriptFunctionality {
    param([string]$ScriptPath, [string]$Description)
    
    Write-ValidationLog "Testing $Description..." "INFO"
    
    try {
        # Test script syntax by importing it
        $scriptContent = Get-Content $ScriptPath -Raw
        $scriptBlock = [ScriptBlock]::Create($scriptContent)
        
        # Basic syntax validation
        if ($scriptBlock) {
            Write-ValidationLog "✅ $Description has valid PowerShell syntax" "SUCCESS"
            return $true
        } else {
            Write-ValidationLog "❌ $Description has syntax errors" "ERROR"
            return $false
        }
    }
    catch {
        Write-ValidationLog "❌ $Description failed validation: $($_.Exception.Message)" "ERROR"  
        return $false
    }
}

function Test-ConfigurationFile {
    Write-ValidationLog "Validating test configuration..." "INFO"
    
    $configPath = "configs/test-config.json"
    if (-not (Test-Path $configPath)) {
        Write-ValidationLog "❌ Test configuration file missing: $configPath" "ERROR"
        return $false
    }
    
    try {
        $config = Get-Content $configPath | ConvertFrom-Json
        
        # Check required sections
        $requiredSections = @("database", "server", "testing", "llm", "rss")
        $allSectionsFound = $true
        
        foreach ($section in $requiredSections) {
            if ($config.$section) {
                Write-ValidationLog "✅ Configuration section found: $section" "SUCCESS"
            } else {
                Write-ValidationLog "❌ Configuration section missing: $section" "ERROR"
                $allSectionsFound = $false
            }
        }
        
        return $allSectionsFound
    }
    catch {
        Write-ValidationLog "❌ Invalid JSON in configuration file: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-DatabaseTestingSetup {
    Write-ValidationLog "Validating database testing setup..." "INFO"
    
    $databaseTestFile = "internal/testing/database.go"
    if (-not (Test-Path $databaseTestFile)) {
        Write-ValidationLog "❌ Database testing file missing: $databaseTestFile" "ERROR"
        return $false
    }
    
    $content = Get-Content $databaseTestFile -Raw
    $requiredFunctions = @(
        "NewSQLiteTestDatabase",
        "NewPostgresTestDatabase", 
        "SeedTestData",
        "CleanupTestData",
        "applyTestSchema"
    )
    
    $allFunctionsFound = $true
    foreach ($func in $requiredFunctions) {
        if ($content.Contains("func $func") -or $content.Contains("func ($func")) {
            Write-ValidationLog "✅ Database function found: $func" "SUCCESS"
        } else {
            Write-ValidationLog "❌ Database function missing: $func" "ERROR"
            $allFunctionsFound = $false
        }
    }
    
    return $allFunctionsFound
}

function Test-ComprehensiveTestRunner {
    Write-ValidationLog "Testing comprehensive test runner..." "INFO"
    
    $testRunnerPath = "scripts/comprehensive-test-runner.ps1"
    if (-not (Test-Path $testRunnerPath)) {
        Write-ValidationLog "❌ Comprehensive test runner missing" "ERROR"
        return $false
    }
    
    # Test dry run with unit tests only
    try {
        $result = & powershell -File $testRunnerPath -TestType unit -SkipServerStart -WhatIf 2>&1
        Write-ValidationLog "✅ Test runner executed successfully (dry run)" "SUCCESS"
        return $true
    }
    catch {
        Write-ValidationLog "❌ Test runner failed: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-GoCompilation {
    Write-ValidationLog "Testing Go compilation..." "INFO"
    
    try {
        $buildOutput = go build ./... 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ValidationLog "✅ Go compilation successful" "SUCCESS"
            return $true
        } else {
            Write-ValidationLog "❌ Go compilation failed:" "ERROR"
            Write-ValidationLog $buildOutput "ERROR"
            return $false
        }
    }
    catch {
        Write-ValidationLog "❌ Go build command failed: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-ContainerSupport {
    if ($SkipContainerTests) {
        Write-ValidationLog "⏭️  Skipping container tests as requested" "INFO"
        return $true
    }
    
    Write-ValidationLog "Testing container support..." "INFO"
    
    # Check if Docker is available
    try {
        $dockerVersion = docker --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ValidationLog "✅ Docker is available: $dockerVersion" "SUCCESS"
            
            # Test Docker daemon connectivity
            $dockerInfo = docker info 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-ValidationLog "✅ Docker daemon is accessible" "SUCCESS"
                return $true
            } else {
                Write-ValidationLog "⚠️  Docker daemon not accessible - container tests may fail" "WARN"
                return $false
            }
        } else {
            Write-ValidationLog "⚠️  Docker not available - container tests will be skipped" "WARN"
            return $false
        }
    }
    catch {
        Write-ValidationLog "⚠️  Docker check failed: $($_.Exception.Message)" "WARN"
        return $false
    }
}

# Main validation execution
function Start-Phase2Validation {
    Write-ValidationLog "=== Phase 2 Implementation Validation ===" "INFO"
    Write-ValidationLog "Validating test infrastructure components..." "INFO"
    
    $validationResults = @()
    
    # File existence checks
    $filesToCheck = @(
        @{Path="scripts/comprehensive-test-runner.ps1"; Description="Comprehensive Test Runner"},
        @{Path="scripts/setup-test-env.ps1"; Description="Test Environment Setup Script"},
        @{Path="internal/testing/database.go"; Description="Database Testing Infrastructure"},
        @{Path="configs/test-config.json"; Description="Test Configuration"},
        @{Path="internal/api/articles_test_example.go"; Description="Example API Test"}
    )
    
    foreach ($file in $filesToCheck) {
        $validationResults += Test-FileExists $file.Path $file.Description
    }
    
    # Functional tests
    $validationResults += Test-ConfigurationFile
    $validationResults += Test-DatabaseTestingSetup
    $validationResults += Test-GoModules
    $validationResults += Test-GoCompilation
    $validationResults += Test-ContainerSupport
    
    # Script functionality tests
    $scriptsToTest = @(
        @{Path="scripts/comprehensive-test-runner.ps1"; Description="Comprehensive Test Runner"},
        @{Path="scripts/setup-test-env.ps1"; Description="Test Environment Setup"}
    )
    
    foreach ($script in $scriptsToTest) {
        $validationResults += Test-ScriptFunctionality $script.Path $script.Description
    }
    
    # Summary
    $passed = ($validationResults | Where-Object { $_ -eq $true }).Count
    $total = $validationResults.Count
    $failed = $total - $passed
    
    Write-ValidationLog "=== Validation Summary ===" "INFO"
    Write-ValidationLog "Total Checks: $total" "INFO"
    Write-ValidationLog "Passed: $passed" "SUCCESS"
    Write-ValidationLog "Failed: $failed" $(if ($failed -eq 0) {"SUCCESS"} else {"ERROR"})
    
    if ($failed -eq 0) {
        Write-ValidationLog "🎉 Phase 2 implementation validation PASSED!" "SUCCESS"
        Write-ValidationLog "All test infrastructure components are properly implemented." "SUCCESS"
        return $true
    } else {
        Write-ValidationLog "❌ Phase 2 implementation validation FAILED!" "ERROR"
        Write-ValidationLog "Please fix the issues above before proceeding to Phase 3." "ERROR"
        return $false
    }
}

# Run validation
try {
    $success = Start-Phase2Validation
    
    if ($success) {
        Write-ValidationLog "Next steps:" "INFO"
        Write-ValidationLog "1. Install missing Go modules if any were reported" "INFO"
        Write-ValidationLog "2. Run: .\scripts\setup-test-env.ps1 -Verbose" "INFO"
        Write-ValidationLog "3. Run: .\scripts\comprehensive-test-runner.ps1 -TestType unit -Verbose" "INFO"
        Write-ValidationLog "4. Proceed to Phase 3: E2E Test Stabilization" "INFO"
        
        exit 0
    } else {
        exit 1
    }
}
catch {
    Write-ValidationLog "Validation script failed: $($_.Exception.Message)" "ERROR"
    exit 1
}
