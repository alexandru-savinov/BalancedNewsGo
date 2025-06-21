# Code Coverage Analysis Script for NewsBalancer (PowerShell)
# This script runs tests with coverage analysis and generates reports

param(
    [string]$MinCoverage = "80",
    [string]$DatabaseUrl = "postgres://localhost:5432/newsbalancer_test?sslmode=disable",
    [string]$OutputDir = "./coverage"
)

$ErrorActionPreference = "Stop"

# Configuration
$CoverageFile = "$OutputDir/coverage.out"
$CoverageHtml = "$OutputDir/coverage.html"
$CoverageJson = "$OutputDir/coverage.json"
$CoverageReport = "$OutputDir/coverage-report.txt"
$TestOutput = "$OutputDir/test-output.log"

Write-Host "NewsBalancer Code Coverage Analysis" -ForegroundColor Blue
Write-Host "===================================" -ForegroundColor Blue
Write-Host ""

# Create coverage directory
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Clean previous coverage data
@($CoverageFile, $CoverageHtml, $CoverageJson, $CoverageReport, $TestOutput) | ForEach-Object {
    if (Test-Path $_) {
        Remove-Item $_ -Force
    }
}

Write-Host "Running tests with coverage..." -ForegroundColor Yellow

# Set environment variables for testing
$env:NO_AUTO_ANALYZE = "true"
$env:DATABASE_URL = $DatabaseUrl

# Run tests with coverage
try {
    $testOutput = go test -v -race -coverprofile="$CoverageFile" -covermode=atomic ./... 2>&1
    $testOutput | Out-File -FilePath $TestOutput -Encoding UTF8
    $testOutput | Write-Host
} catch {
    Write-Host "Error running tests: $_" -ForegroundColor Red
    exit 1
}

if (!(Test-Path $CoverageFile)) {
    Write-Host "Error: Coverage file not generated" -ForegroundColor Red
    exit 1
}

Write-Host "Tests completed successfully" -ForegroundColor Green
Write-Host ""

# Generate HTML coverage report
Write-Host "Generating HTML coverage report..." -ForegroundColor Yellow
go tool cover -html="$CoverageFile" -o "$CoverageHtml"
Write-Host "HTML report generated: $CoverageHtml" -ForegroundColor Green

# Generate detailed coverage analysis
Write-Host "Analyzing coverage by package..." -ForegroundColor Yellow

# Create a detailed coverage report
$reportContent = @"
# NewsBalancer Code Coverage Report
Generated: $(Get-Date)

## Overall Coverage
"@

# Calculate overall coverage
$coverageOutput = go tool cover -func="$CoverageFile"
$totalLine = $coverageOutput | Where-Object { $_ -match "total:" }
if ($totalLine) {
    $totalCoverage = ($totalLine -split '\s+')[2] -replace '%', ''
    $reportContent += "`nTotal Coverage: ${totalCoverage}%`n"
} else {
    Write-Host "Warning: Could not parse total coverage" -ForegroundColor Yellow
    $totalCoverage = "0"
}

$reportContent += "`n## Package Coverage`n"

# Package-level coverage
$packageLines = $coverageOutput | Where-Object { $_ -match "\.go:" -and $_ -notmatch "total:" }
foreach ($line in $packageLines) {
    if ($line -match "(.+\.go):.*\s+(\d+\.\d+%)") {
        $package = $matches[1]
        $coverage = $matches[2]
        $reportContent += "- $package`: $coverage`n"
    }
}

# Functions with low coverage
$reportContent += "`n## Functions with Low Coverage (<50%)`n"
foreach ($line in $packageLines) {
    if ($line -match "(.+)\s+(\d+\.\d+%)$") {
        $func = $matches[1]
        $coverage = [double]($matches[2] -replace '%', '')
        if ($coverage -lt 50 -and $coverage -gt 0) {
            $reportContent += "- $func`: $($matches[2])`n"
        }
    }
}

$reportContent | Out-File -FilePath $CoverageReport -Encoding UTF8

# Generate JSON report
Write-Host "Generating JSON coverage report..." -ForegroundColor Yellow

$coveragePassed = [double]$totalCoverage -ge [double]$MinCoverage

$jsonReport = @{
    timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
    total_coverage = [double]$totalCoverage
    threshold = [double]$MinCoverage
    passed = $coveragePassed
    packages = @()
}

# Add package coverage to JSON
foreach ($line in $packageLines) {
    if ($line -match "(.+\.go):.*\s+(\d+\.\d+%)") {
        $package = $matches[1]
        $coverage = [double]($matches[2] -replace '%', '')
        $jsonReport.packages += @{
            package = $package
            coverage = $coverage
        }
    }
}

$jsonReport | ConvertTo-Json -Depth 3 | Out-File -FilePath $CoverageJson -Encoding UTF8
Write-Host "JSON report generated: $CoverageJson" -ForegroundColor Green
Write-Host ""

# Display summary
Write-Host "Coverage Summary:" -ForegroundColor Blue
Write-Host "  Total Coverage: ${totalCoverage}%"
Write-Host "  Minimum Required: ${MinCoverage}%"

if ($coveragePassed) {
    Write-Host "  Status: PASSED" -ForegroundColor Green
    $exitCode = 0
} else {
    Write-Host "  Status: FAILED" -ForegroundColor Red
    Write-Host "  Coverage is below the minimum threshold of ${MinCoverage}%" -ForegroundColor Red
    $exitCode = 1
}

Write-Host ""
Write-Host "Generated Files:" -ForegroundColor Blue
Write-Host "  Coverage Data: $CoverageFile"
Write-Host "  HTML Report: $CoverageHtml"
Write-Host "  JSON Report: $CoverageJson"
Write-Host "  Text Report: $CoverageReport"
Write-Host "  Test Output: $TestOutput"

Write-Host ""
Write-Host "To view the HTML report, open: file://$(Resolve-Path $CoverageHtml)" -ForegroundColor Yellow

# Check if we should fail based on coverage
if ($exitCode -ne 0) {
    Write-Host ""
    Write-Host "Coverage check failed. Please improve test coverage before proceeding." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Coverage analysis completed successfully!" -ForegroundColor Green
