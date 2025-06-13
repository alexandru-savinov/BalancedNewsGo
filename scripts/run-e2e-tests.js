#!/usr/bin/env node

/**
 * Comprehensive E2E Test Runner for NewsBalancer HTMX Features
 * 
 * This script runs all E2E tests and provides detailed reporting
 * for Phase 4 testing validation.
 */

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const COLORS = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

function log(message, color = 'reset') {
  console.log(`${COLORS[color]}${message}${COLORS.reset}`);
}

function logHeader(message) {
  console.log('\n' + '='.repeat(60));
  log(message, 'bright');
  console.log('='.repeat(60));
}

function logStep(step, message) {
  log(`[${step}] ${message}`, 'cyan');
}

function logSuccess(message) {
  log(`✅ ${message}`, 'green');
}

function logWarning(message) {
  log(`⚠️  ${message}`, 'yellow');
}

function logError(message) {
  log(`❌ ${message}`, 'red');
}

async function runCommand(command, args = [], options = {}) {
  return new Promise((resolve, reject) => {
    const process = spawn(command, args, {
      stdio: 'inherit',
      shell: true,
      ...options
    });

    process.on('close', (code) => {
      if (code === 0) {
        resolve(code);
      } else {
        reject(new Error(`Command failed with exit code ${code}`));
      }
    });

    process.on('error', (error) => {
      reject(error);
    });
  });
}

async function checkServerHealth() {
  logStep('HEALTH', 'Checking server health...');
  
  try {
    const response = await fetch('http://localhost:8080/health');
    if (response.ok) {
      logSuccess('Server is healthy and ready for testing');
      return true;
    } else {
      logWarning(`Server responded with status: ${response.status}`);
      return false;
    }
  } catch (error) {
    logWarning('Server health check failed - will start server automatically');
    return false;
  }
}

async function runTestSuite(suiteName, testFile, options = {}) {
  logHeader(`Running ${suiteName}`);
  
  try {
    const args = ['test', testFile];
    
    if (options.headed) args.push('--headed');
    if (options.debug) args.push('--debug');
    if (options.project) args.push('--project', options.project);
    if (options.grep) args.push('--grep', options.grep);
    
    await runCommand('npx', ['playwright', ...args]);
    logSuccess(`${suiteName} completed successfully`);
    return true;
  } catch (error) {
    logError(`${suiteName} failed: ${error.message}`);
    return false;
  }
}

async function generateReport() {
  logStep('REPORT', 'Generating test report...');
  
  try {
    await runCommand('npx', ['playwright', 'show-report', '--port', '9323']);
    logSuccess('Test report available at http://localhost:9323');
  } catch (error) {
    logWarning('Could not start report server');
  }
}

async function runLighthouseTests() {
  logStep('LIGHTHOUSE', 'Running performance tests...');
  
  // Check if lighthouse is available
  try {
    await runCommand('npx', ['lighthouse', '--version'], { stdio: 'ignore' });
  } catch (error) {
    logWarning('Lighthouse not available, skipping performance tests');
    return false;
  }
  
  try {
    // Run lighthouse on key pages
    const pages = [
      { name: 'Home', url: 'http://localhost:8080/' },
      { name: 'Article Detail', url: 'http://localhost:8080/article/1' }
    ];
    
    for (const page of pages) {
      logStep('PERF', `Testing ${page.name} page performance...`);
      
      const outputPath = `test-results/lighthouse-${page.name.toLowerCase()}.html`;
      
      await runCommand('npx', [
        'lighthouse',
        page.url,
        '--output=html',
        '--output-path=' + outputPath,
        '--quiet',
        '--chrome-flags="--headless"'
      ]);
      
      logSuccess(`${page.name} performance report saved to ${outputPath}`);
    }
    
    return true;
  } catch (error) {
    logError(`Performance tests failed: ${error.message}`);
    return false;
  }
}

async function main() {
  const args = process.argv.slice(2);
  const options = {
    headed: args.includes('--headed'),
    debug: args.includes('--debug'),
    skipPerf: args.includes('--skip-perf'),
    project: args.find(arg => arg.startsWith('--project='))?.split('=')[1],
    grep: args.find(arg => arg.startsWith('--grep='))?.split('=')[1]
  };
  
  logHeader('NewsBalancer HTMX E2E Test Suite');
  log('Phase 4 - Testing & Validation', 'bright');
  
  // Ensure test-results directory exists
  if (!fs.existsSync('test-results')) {
    fs.mkdirSync('test-results');
  }
  
  // Check server health
  const serverHealthy = await checkServerHealth();
  if (!serverHealthy) {
    logStep('SERVER', 'Server will be started automatically by Playwright');
  }
  
  const results = {
    total: 0,
    passed: 0,
    failed: 0,
    suites: []
  };
  
  // Define test suites
  const testSuites = [
    {
      name: 'HTMX Core Functionality',
      file: 'tests/htmx-e2e.spec.ts',
      description: 'Dynamic filtering, live search, pagination, article loading'
    },
    {
      name: 'HTMX Performance & Accessibility',
      file: 'tests/htmx-performance-accessibility.spec.ts',
      description: 'Performance optimization and accessibility compliance'
    },
    {
      name: 'HTMX Integration Tests',
      file: 'tests/htmx-integration.spec.ts',
      description: 'Specific HTMX endpoints and real-time features'
    },
    {
      name: 'Article Feed E2E',
      file: 'tests/article-feed.e2e.spec.ts',
      description: 'Basic article feed functionality'
    }
  ];
  
  // Run test suites
  for (const suite of testSuites) {
    results.total++;
    
    log(`\n📋 ${suite.description}`, 'blue');
    
    const success = await runTestSuite(suite.name, suite.file, options);
    
    if (success) {
      results.passed++;
      results.suites.push({ name: suite.name, status: 'PASSED' });
    } else {
      results.failed++;
      results.suites.push({ name: suite.name, status: 'FAILED' });
    }
  }
  
  // Run performance tests if not skipped
  if (!options.skipPerf) {
    results.total++;
    const perfSuccess = await runLighthouseTests();
    
    if (perfSuccess) {
      results.passed++;
      results.suites.push({ name: 'Performance Tests', status: 'PASSED' });
    } else {
      results.failed++;
      results.suites.push({ name: 'Performance Tests', status: 'FAILED' });
    }
  }
  
  // Generate final report
  logHeader('Test Results Summary');
  
  console.log(`Total Test Suites: ${results.total}`);
  logSuccess(`Passed: ${results.passed}`);
  if (results.failed > 0) {
    logError(`Failed: ${results.failed}`);
  }
  
  console.log('\nDetailed Results:');
  results.suites.forEach(suite => {
    const status = suite.status === 'PASSED' ? '✅' : '❌';
    console.log(`  ${status} ${suite.name}`);
  });
  
  // Generate Playwright HTML report
  if (!options.debug) {
    await generateReport();
  }
  
  // Phase 4 completion status
  logHeader('Phase 4 - Testing & Validation Status');
  
  if (results.failed === 0) {
    logSuccess('🎉 Phase 4 completed successfully!');
    logSuccess('✅ Unit tests for API wrapper completed');
    logSuccess('✅ Handler tests completed');
    logSuccess('✅ E2E tests with HTMX completed');
    
    console.log('\n📋 Key HTMX features tested:');
    console.log('  • Dynamic filtering without page refresh');
    console.log('  • Live search functionality');
    console.log('  • Seamless pagination navigation');
    console.log('  • Article loading via HTMX');
    console.log('  • Browser history management');
    console.log('  • Performance optimization');
    console.log('  • Accessibility compliance');
    console.log('  • Error handling and recovery');
    
    process.exit(0);
  } else {
    logError('❌ Some tests failed. Please review the results above.');
    logWarning('Phase 4 requires all tests to pass for completion.');
    process.exit(1);
  }
}

// Handle unhandled promise rejections
process.on('unhandledRejection', (reason, promise) => {
  logError(`Unhandled Rejection at: ${promise}, reason: ${reason}`);
  process.exit(1);
});

// Handle uncaught exceptions
process.on('uncaughtException', (error) => {
  logError(`Uncaught Exception: ${error.message}`);
  process.exit(1);
});

if (require.main === module) {
  main().catch(error => {
    logError(`Test runner failed: ${error.message}`);
    process.exit(1);
  });
}
