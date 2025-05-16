# NewsBalancer Database Reset Script
# This script recreates the database with proper schema

Write-Host "=== NEWSBALANCER DATABASE RESET SCRIPT ===" -ForegroundColor Cyan
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

# Create backup directory
Write-Host "1. Creating backup directory..." -ForegroundColor Green
New-Item -Path "backup" -ItemType Directory -Force | Out-Null

# Backup current database
Write-Host "2. Backing up existing database files..." -ForegroundColor Green
if (Test-Path "news.db") {
    Copy-Item "news.db" "backup/news.db.backup_$timestamp"
    Write-Host "   ✓ Backed up news.db" -ForegroundColor Gray
}

# Stop any running processes
Write-Host "3. Stopping any conflicting processes..." -ForegroundColor Green
Stop-Process -Name "go", "server", "newbalancer" -Force -ErrorAction SilentlyContinue

# Remove database files
Write-Host "4. Removing old database files..." -ForegroundColor Green
Remove-Item "news.db", "news.db-shm", "news.db-wal" -Force -ErrorAction SilentlyContinue

# Recreate the database
Write-Host "5. Recreating database..." -ForegroundColor Green
go run ./cmd/reset_test_db
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✓ Database recreated successfully" -ForegroundColor Green
} else {
    Write-Host "   ✗ Failed to recreate database" -ForegroundColor Red
    exit 1
}

# Verify database exists
if (Test-Path "news.db") {
    Write-Host "6. Database file verified" -ForegroundColor Green
} else {
    Write-Host "   ✗ Database file not created" -ForegroundColor Red
    exit 1
}

# Run verification
Write-Host "7. Running integrity check..." -ForegroundColor Green
$integrity = sqlite3 news.db "PRAGMA integrity_check;"
if ($integrity -match "ok") {
    Write-Host "   ✓ Database integrity verified" -ForegroundColor Green
} else {
    Write-Host "   ✗ Database integrity check failed" -ForegroundColor Red
    Write-Host $integrity -ForegroundColor Red
}

Write-Host "`n✅ DATABASE RESET COMPLETED SUCCESSFULLY" -ForegroundColor Green
Write-Host "   Original database backed up to: backup/news.db.backup_$timestamp" -ForegroundColor Gray
Write-Host "   Run 'scripts/test.cmd essential' to verify functionality" -ForegroundColor Gray 