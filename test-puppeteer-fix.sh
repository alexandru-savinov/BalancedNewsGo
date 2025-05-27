#!/bin/bash

# Puppeteer Navigation Fix Test Script
# This script tests the fixed puppeteer navigation implementation

echo "🔧 Testing Puppeteer Navigation Fix"
echo "=================================="

# Check if we're in the right directory
if [ ! -f "package.json" ]; then
    echo "❌ Error: package.json not found. Please run this script from the project root."
    exit 1
fi

# Check if puppeteer is installed
echo "📦 Checking Puppeteer installation..."
if npm list puppeteer >/dev/null 2>&1; then
    echo "✅ Puppeteer is installed"
else
    echo "⚠ Puppeteer not found, installing..."
    npm install puppeteer
fi

# Check if the server is running
echo "🌐 Checking if NewsBalancer server is running..."
if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "✅ Server is running on port 8080"
else
    echo "⚠ Server not detected on port 8080"
    echo "💡 Please start the NewsBalancer server before running tests"
    echo "   You can start it with: go run cmd/server/main.go"
fi

# Navigate to tests directory
cd web/tests

echo "🧪 Running individual test modules..."

# Test the helper class first
echo "📋 Testing Puppeteer Helper..."
node -e "
const PuppeteerHelper = require('./puppeteer-helper');
async function testHelper() {
    const helper = new PuppeteerHelper();
    try {
        await helper.initialize({ headless: true });
        console.log('✅ PuppeteerHelper initialization successful');
        await helper.navigate('http://localhost:8080');
        console.log('✅ Navigation test successful');
        await helper.cleanup();
        console.log('✅ Cleanup successful');
    } catch (error) {
        console.error('❌ Helper test failed:', error.message);
        process.exit(1);
    }
}
testHelper();
"

echo "🧪 Running filter tests..."
if node filter.test.js; then
    echo "✅ Filter tests passed"
else
    echo "❌ Filter tests failed"
fi

echo "🧪 Running homepage tests..."
if node homepage.test.js; then
    echo "✅ Homepage tests passed"
else
    echo "❌ Homepage tests failed"
fi

echo "🧪 Running article detail tests..."
if node article-detail.test.js; then
    echo "✅ Article detail tests passed"
else
    echo "❌ Article detail tests failed"
fi

echo "🚀 Running complete test suite..."
node run-tests.js

echo ""
echo "🎯 Puppeteer navigation fix testing complete!"
echo "   If tests are passing, the 'detached Frame' issue has been resolved."
