@echo off
REM Puppeteer Navigation Fix Test Script for Windows
REM This script tests the fixed puppeteer navigation implementation

echo 🔧 Testing Puppeteer Navigation Fix
echo ==================================

REM Check if we're in the right directory
if not exist "package.json" (
    echo ❌ Error: package.json not found. Please run this script from the project root.
    exit /b 1
)

REM Check if puppeteer is installed
echo 📦 Checking Puppeteer installation...
npm list puppeteer >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Puppeteer is installed
) else (
    echo ⚠ Puppeteer not found, installing...
    npm install puppeteer
)

REM Check if the server is running
echo 🌐 Checking if NewsBalancer server is running...
curl -s http://localhost:8080/health >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Server is running on port 8080
) else (
    echo ⚠ Server not detected on port 8080
    echo 💡 Please start the NewsBalancer server before running tests
    echo    You can start it with: go run cmd/server/main.go
)

REM Navigate to tests directory
cd web\tests

echo 🧪 Running individual test modules...

REM Test the helper class first
echo 📋 Testing Puppeteer Helper...
node -e "const PuppeteerHelper = require('./puppeteer-helper'); async function testHelper() { const helper = new PuppeteerHelper(); try { await helper.initialize({ headless: true }); console.log('✅ PuppeteerHelper initialization successful'); await helper.navigate('http://localhost:8080'); console.log('✅ Navigation test successful'); await helper.cleanup(); console.log('✅ Cleanup successful'); } catch (error) { console.error('❌ Helper test failed:', error.message); process.exit(1); } } testHelper();"

echo 🧪 Running filter tests...
node filter.test.js
if %errorlevel% equ 0 (
    echo ✅ Filter tests passed
) else (
    echo ❌ Filter tests failed
)

echo 🧪 Running homepage tests...
node homepage.test.js
if %errorlevel% equ 0 (
    echo ✅ Homepage tests passed
) else (
    echo ❌ Homepage tests failed
)

echo 🧪 Running article detail tests...
node article-detail.test.js
if %errorlevel% equ 0 (
    echo ✅ Article detail tests passed
) else (
    echo ❌ Article detail tests failed
)

echo 🚀 Running complete test suite...
node run-tests.js

echo.
echo 🎯 Puppeteer navigation fix testing complete!
echo    If tests are passing, the 'detached Frame' issue has been resolved.
