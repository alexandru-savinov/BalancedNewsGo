module.exports = {
  // Common test settings
  baseUrl: 'http://localhost:8080',
  timeout: 30000,
  
  // Test data management
  testData: {
    snapshotDir: './e2e_snapshots',
    resultsDir: './test-results',
    cleanupOlderThan: '7d'
  },
  
  // Framework-specific configs
  playwright: {
    browser: 'chromium',
    headless: true,
    viewport: { width: 1280, height: 720 }
  },
  
  newman: {
    reporters: ['cli', 'json', 'html'],
    iterationCount: 1,
    delayRequest: 100
  },

  // Test categorization
  suites: {
    unit: {
      dir: './tests/unit',
      pattern: '**/*.test.{js,ts}'
    },
    integration: {
      dir: './tests/integration',
      pattern: '**/*.test.{js,ts}'  
    },
    e2e: {
      dir: './tests/e2e',
      pattern: '**/*.spec.{js,ts}'
    }
  }
}