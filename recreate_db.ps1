#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Recreates the NewsBalancer database to fix corruption issues.
.DESCRIPTION
    This script performs a complete recreation of the NewsBalancer SQLite database:
    1. Backs up existing database files
    2. Stops any processes that might be using the database
    3. Deletes corrupted database files
    4. Recreates the database using the application's built-in InitDB function
    5. Verifies the database creation and schema integrity
    6. Runs essential tests to confirm functionality
.NOTES
    Requires: PowerShell 5.1+, SQLite3 CLI, Go
#>

# Set strict error handling
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

Write-Host "=== NEWSBALANCER DATABASE RECREATION PROCESS - $timestamp ===" -ForegroundColor Cyan

function Test-CommandExists {
    param ($command)
    $null -ne (Get-Command $command -ErrorAction SilentlyContinue)
}

try {
    # Check prerequisites
    if (-not (Test-CommandExists "sqlite3")) {
        throw "SQLite3 command-line tool is not installed or not in PATH. Please install it first."
    }

    if (-not (Test-CommandExists "go")) {
        throw "Go is not installed or not in PATH. Please install it first."
    }

    # 1. Create backup directory
    Write-Host "1. Creating backup directory..." -ForegroundColor Green
    $null = New-Item -Path "backup" -ItemType Directory -Force

    # 2. Backup current database files with timestamp
    Write-Host "2. Backing up current database files..." -ForegroundColor Green
    if (Test-Path "news.db") {
        Copy-Item "news.db" "backup/news.db.backup_$timestamp"
        Write-Host "   ✓ Backed up news.db" -ForegroundColor Gray
    } else {
        Write-Host "   ! No news.db found to backup" -ForegroundColor Yellow
    }

    if (Test-Path "news.db-shm") {
        Copy-Item "news.db-shm" "backup/news.db-shm.backup_$timestamp"
        Write-Host "   ✓ Backed up news.db-shm" -ForegroundColor Gray
    }

    if (Test-Path "news.db-wal") {
        Copy-Item "news.db-wal" "backup/news.db-wal.backup_$timestamp"
        Write-Host "   ✓ Backed up news.db-wal" -ForegroundColor Gray
    }

    # 3. Stop any running services that might be using the database
    Write-Host "3. Stopping any running services..." -ForegroundColor Green
    $processes = @("go", "server", "newbalancer")
    foreach ($proc in $processes) {
        $runningProcesses = Get-Process -Name $proc -ErrorAction SilentlyContinue
        if ($runningProcesses) {
            foreach ($p in $runningProcesses) {
                Stop-Process -Id $p.Id -Force
                Write-Host "   ✓ Stopped process: $($p.ProcessName) (PID: $($p.Id))" -ForegroundColor Gray
            }
        }
    }

    # 4. Remove corrupted database files
    Write-Host "4. Removing corrupted database files..." -ForegroundColor Green
    $dbFiles = @("news.db", "news.db-shm", "news.db-wal")
    foreach ($file in $dbFiles) {
        if (Test-Path $file) {
            Remove-Item $file -Force
            Write-Host "   ✓ Removed $file" -ForegroundColor Gray
        }
    }

    # 5. Recreate database using the built-in InitDB function
    Write-Host "5. Recreating database schema using built-in InitDB function..." -ForegroundColor Green
    $output = go run ./cmd/reset_test_db 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to recreate database: $output"
    }
    Write-Host "   ✓ Database recreated successfully" -ForegroundColor Gray

    # 6. Verify database creation and schema
    Write-Host "6. Verifying database creation and schema..." -ForegroundColor Green
    if (-not (Test-Path "news.db")) {
        throw "Database file 'news.db' was not created"
    }

    # 6a. Check database integrity
    $integrity = sqlite3 news.db "PRAGMA integrity_check;"
    if ($integrity -notmatch "ok") {
        Write-Host "   ! Database integrity check failed:" -ForegroundColor Red
        Write-Host $integrity -ForegroundColor Red
        throw "Database integrity check failed"
    }
    Write-Host "   ✓ Database integrity verified" -ForegroundColor Gray

    # 6b. Verify tables exist
    $tables = sqlite3 news.db ".tables"
    $requiredTables = @("articles", "llm_scores", "feedback", "labels")
    foreach ($table in $requiredTables) {
        if ($tables -notmatch $table) {
            throw "Missing required table: $table"
        }
    }
    Write-Host "   ✓ All required tables exist" -ForegroundColor Gray

    # 6c. Verify UNIQUE constraint on llm_scores
    $constraints = sqlite3 news.db "SELECT sql FROM sqlite_master WHERE type='table' AND name='llm_scores';"
    if ($constraints -notmatch "UNIQUE\s*\(\s*article_id\s*,\s*model\s*\)") {
        throw "Missing UNIQUE(article_id, model) constraint on llm_scores table"
    }
    Write-Host "   ✓ Critical UNIQUE constraint exists on llm_scores" -ForegroundColor Gray

    # 7. Run tests to verify database functionality
    Write-Host "7. Running essential tests to verify database functionality..." -ForegroundColor Green
    $env:NO_AUTO_ANALYZE = 'true'
    $testOutput = $null
    try {
        $testOutput = scripts/test.cmd essential 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "   ! Essential tests failed. Review test output:" -ForegroundColor Red
            Write-Host $testOutput -ForegroundColor Red
            throw "Essential tests failed"
        }
        Write-Host "   ✓ Essential tests passed" -ForegroundColor Gray
    }
    catch {
        Write-Host "   ! Error running tests: $($_.Exception.Message)" -ForegroundColor Red
        throw $_
    }
    finally {
        # Always write the test results
        if ($testOutput) {
            $testOutput | Out-File -FilePath "backup/db_recreation_test_results_$timestamp.log"
            Write-Host "   ✓ Test results saved to backup/db_recreation_test_results_$timestamp.log" -ForegroundColor Gray
        }
    }

    # 8. Success message
    Write-Host "`n✅ DATABASE RECREATION COMPLETED SUCCESSFULLY" -ForegroundColor Green
    Write-Host "   Original database backed up to: backup/news.db.backup_$timestamp" -ForegroundColor Gray
    Write-Host "   New database created with proper schema and verified with tests" -ForegroundColor Gray
    Write-Host "   Environment ready for further development and testing" -ForegroundColor Gray
    exit 0
}
catch {
    # Handle errors
    Write-Host "`n❌ ERROR: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   Process failed at step: $($_.InvocationInfo.ScriptLineNumber)" -ForegroundColor Red
    Write-Host "   See backup directory for any saved database files" -ForegroundColor Red
    Write-Host "   You may need to manually restore from backup/news.db.backup_$timestamp if needed" -ForegroundColor Red
    exit 1
}
