import { defineConfig, devices } from '@playwright/test';

/**
 * Enhanced Playwright Configuration for HTMX Applications
 * Phase 3: E2E Test Stabilization Implementation
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  // Global test timeout
  timeout: 30000,
  expect: {
    // Timeout for assertions
    timeout: 10000,
    // Screenshot comparison tolerance
    toHaveScreenshot: { maxDiffPixels: 100 },
    toMatchSnapshot: { maxDiffPixels: 100 }
  },
  
  // Test directory
  testDir: './tests',
    // Folder for test artifacts such as screenshots, videos, traces, etc.
  outputDir: 'test-results/artifacts',
  
  // Run tests in files in parallel
  fullyParallel: true,
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only
  retries: process.env.CI ? 2 : 0,
  
  // Opt out of parallel tests on CI
  workers: process.env.CI ? 1 : undefined,
  
  // Reporter configuration
  reporter: [
    ['html', { outputFolder: 'test-results/playwright-report' }],
    ['json', { outputFile: 'test-results/test-results.json' }],
    ['junit', { outputFile: 'test-results/junit.xml' }]
  ],
  
  // Shared settings for all tests
  use: {
    // Base URL for navigation
    baseURL: 'http://localhost:8080',
    
    // Take screenshot on failure
    screenshot: 'only-on-failure',
    
    // Record video on failure
    video: 'retain-on-failure',
    
    // Record trace on failure for debugging
    trace: 'retain-on-failure',
    
    // Browser context settings
    viewport: { width: 1280, height: 720 },
    
    // Ignore HTTPS errors
    ignoreHTTPSErrors: true,
    
    // Wait for network to be idle before proceeding
    actionTimeout: 15000
  },
  // Cross-browser testing projects
  projects: [
    // Desktop browsers
    {
      name: 'chromium',
      use: { 
        ...devices['Desktop Chrome'],
      },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    
    // Mobile browsers
    {
      name: 'Mobile Chrome',
      use: { 
        ...devices['Pixel 5'],
        hasTouch: true
      },
    },
    {
      name: 'Mobile Safari',
      use: { 
        ...devices['iPhone 12'],
        hasTouch: true
      },
    },
    
    // Branded browsers for compatibility testing
    {
      name: 'Google Chrome',
      use: { 
        ...devices['Desktop Chrome'], 
        channel: 'chrome' 
      },
    },
    {
      name: 'Microsoft Edge',
      use: { 
        ...devices['Desktop Edge'], 
        channel: 'msedge' 
      },
    },
  ],

  // Global setup and teardown
  // globalSetup: require.resolve('./tests/global-setup.ts'),
  // globalTeardown: require.resolve('./tests/global-teardown.ts'),
  
  // Web server configuration - only start server if not in CI (CI starts server manually)
  webServer: process.env.CI ? undefined : {
    command: 'go run ./cmd/server',
    port: 8080,
    timeout: 120 * 1000, // 2 minutes
    reuseExistingServer: true,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
