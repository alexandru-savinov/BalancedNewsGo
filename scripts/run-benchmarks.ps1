# NewsBalancer Performance Benchmark Script (PowerShell)
# This script runs various performance benchmarks and generates reports

param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$DbUrl = "",
    [string]$OutputDir = "./benchmark-results"
)

$ErrorActionPreference = "Stop"

# Configuration
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

Write-Host "NewsBalancer Performance Benchmark Suite" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue
Write-Host ""

# Create output directory
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Check if the API is running
Write-Host "Checking API availability..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/api/articles" -Method GET -TimeoutSec 10
    Write-Host "API is accessible" -ForegroundColor Green
} catch {
    Write-Host "Error: API is not accessible at $BaseUrl" -ForegroundColor Red
    Write-Host "Please ensure the NewsBalancer server is running"
    exit 1
}
Write-Host ""

# Build benchmark tool if it doesn't exist
$BenchmarkPath = "./cmd/benchmark/benchmark.exe"
if (!(Test-Path $BenchmarkPath)) {
    Write-Host "Building benchmark tool..." -ForegroundColor Yellow
    Push-Location "cmd/benchmark"
    try {
        go build -o benchmark.exe .
        Write-Host "Benchmark tool built" -ForegroundColor Green
    } catch {
        Write-Host "Failed to build benchmark tool: $_" -ForegroundColor Red
        exit 1
    } finally {
        Pop-Location
    }
}

# Function to run a benchmark test
function Run-Benchmark {
    param(
        [string]$TestName,
        [int]$Users,
        [int]$Requests,
        [string]$Duration
    )
    
    $OutputFile = "$OutputDir/${TestName}_${Timestamp}"
    
    Write-Host "Running $TestName benchmark..." -ForegroundColor Yellow
    Write-Host "  Users: $Users, Requests per user: $Requests, Duration: $Duration"
    
    # Run benchmark and save results
    try {
        & $BenchmarkPath -test $TestName -url $BaseUrl -users $Users -requests $Requests -duration $Duration -db $DbUrl -output json | Out-File -FilePath "${OutputFile}.json" -Encoding UTF8
        & $BenchmarkPath -test $TestName -url $BaseUrl -users $Users -requests $Requests -duration $Duration -db $DbUrl -output csv | Out-File -FilePath "${OutputFile}.csv" -Encoding UTF8
        
        Write-Host "$TestName benchmark completed" -ForegroundColor Green
        Write-Host "  Results saved to: ${OutputFile}.json and ${OutputFile}.csv"
    } catch {
        Write-Host "Failed to run $TestName benchmark: $_" -ForegroundColor Red
    }
    Write-Host ""
}

# Run different benchmark scenarios
Write-Host "Starting benchmark tests..." -ForegroundColor Blue
Write-Host ""

# Light load test
Run-Benchmark -TestName "light-load" -Users 5 -Requests 20 -Duration "2m"

# Medium load test
Run-Benchmark -TestName "medium-load" -Users 15 -Requests 50 -Duration "3m"

# Heavy load test
Run-Benchmark -TestName "heavy-load" -Users 30 -Requests 100 -Duration "5m"

# Stress test
Run-Benchmark -TestName "stress-test" -Users 50 -Requests 200 -Duration "10m"

# Spike test (quick burst)
Run-Benchmark -TestName "spike-test" -Users 100 -Requests 10 -Duration "1m"

# Generate summary report
Write-Host "Generating summary report..." -ForegroundColor Blue
$SummaryFile = "$OutputDir/benchmark_summary_${Timestamp}.md"

$SummaryContent = @"
# NewsBalancer Performance Benchmark Report

**Generated:** $(Get-Date)
**API URL:** $BaseUrl
**Test Suite:** Automated Performance Benchmarks

## Test Results Summary

"@

# Process JSON results to create summary
$JsonFiles = Get-ChildItem -Path $OutputDir -Filter "*_${Timestamp}.json"
foreach ($JsonFile in $JsonFiles) {
    $TestName = $JsonFile.BaseName -replace "_${Timestamp}", ""
    
    try {
        $JsonContent = Get-Content $JsonFile.FullName | ConvertFrom-Json
        
        $SummaryContent += @"

### $TestName

- **Total Requests:** $($JsonContent.total_requests)
- **Successful Requests:** $($JsonContent.successful_requests)
- **Error Rate:** $($JsonContent.error_rate)%
- **Average Latency:** $($JsonContent.average_latency)
- **95th Percentile Latency:** $($JsonContent.p95_latency)
- **Requests per Second:** $($JsonContent.requests_per_second)

"@
    } catch {
        $SummaryContent += @"

### $TestName

*(Error parsing results - see JSON file for detailed results)*

"@
    }
}

$SummaryContent += @"

## Files Generated

"@

# List all generated files
$GeneratedFiles = Get-ChildItem -Path $OutputDir -Filter "*_${Timestamp}.*"
foreach ($File in $GeneratedFiles) {
    $SummaryContent += "- ``$($File.Name)``" + "`n"
}

$SummaryContent += @"

## Recommendations

Based on the benchmark results:

1. **Performance Baseline:** Use these results as a baseline for future performance comparisons
2. **Monitoring:** Set up alerts based on the 95th percentile latency values
3. **Scaling:** Consider horizontal scaling if error rates exceed 1% under normal load
4. **Optimization:** Focus on endpoints with highest latency for optimization efforts

## Next Steps

1. Review individual test results in the JSON/CSV files
2. Compare results with previous benchmark runs
3. Investigate any performance regressions
4. Update monitoring thresholds based on these results

"@

$SummaryContent | Out-File -FilePath $SummaryFile -Encoding UTF8

Write-Host "Summary report generated: $SummaryFile" -ForegroundColor Green
Write-Host ""

# Display quick summary
Write-Host "Quick Summary:" -ForegroundColor Blue
Write-Host "  Output directory: $OutputDir"
Write-Host "  Files generated: $($GeneratedFiles.Count)"
Write-Host "  Summary report: $SummaryFile"
Write-Host ""

Write-Host "All benchmarks completed successfully!" -ForegroundColor Green
Write-Host "Review the generated files for detailed performance analysis." -ForegroundColor Yellow
