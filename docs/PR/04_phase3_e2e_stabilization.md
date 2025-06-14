# Phase 3: E2E Test Stabilization Implementation Guide
*Priority*: P1 - HIGH | *Estimated Time*: 1-2 hours | *Dependencies*: Phase 1 + 2 Complete

## 📋 Phase Overview

**Objective**: Stabilize E2E tests with reliable HTMX functionality validation and cross-browser compatibility  
**Success Criteria**: E2E tests achieve >85% pass rate across all browsers, HTMX interactions work consistently  
**Next Phase**: Proceed to `05_phase4_process_integration.md`

### What This Phase Accomplishes
- **✅ HTMX Test Reliability**: Proper testing of dynamic content loading and interactions
- **✅ Cross-Browser Compatibility**: Consistent behavior across Chromium, Firefox, WebKit
- **✅ Mobile Responsiveness**: Validated functionality on mobile devices
- **✅ Accessibility Standards**: WCAG 2.1 AA compliance verified
- **✅ Performance Validation**: Page load times and interaction performance measured

## 🎭 Current Failure Analysis

### Failed Test Breakdown (132 total failures)
```
Dynamic Content Loading (HTMX) - 72 tests failed
├── Dynamic Filtering: 12 tests  
├── Live Search: 12 tests
├── Pagination: 12 tests
├── Article Loading: 12 tests
├── History Management: 12 tests
└── HTMX Features: 12 tests

Basic Functionality - 18 tests failed
├── Article Feed: 6 tests
├── Navigation: 6 tests  
└── API Integration: 6 tests

Integration Tests - 30 tests failed
├── HTMX Integration: 24 tests
└── Article List: 6 tests

Performance & Accessibility - 12 tests failed
├── Performance Budget: 6 tests
└── ARIA Attributes: 6 tests
```

**Root Cause**: Server not running during E2E test execution, causing all HTMX and API requests to fail.

## 🛠️ Implementation Steps

### Step 3.1: Enhanced HTMX Testing Framework *(45 minutes)*

#### Playwright Configuration for HTMX Applications

**File**: `playwright.config.ts`

```typescript
import { defineConfig, devices } from '@playwright/test';

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
  testDir: './tests/e2e',
  
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
        // Enable console logs for debugging
        launchOptions: {
          args: ['--enable-logging', '--v=1']
        }
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
        isMobile: true,
        hasTouch: true
      },
    },
    {
      name: 'Mobile Safari',
      use: { 
        ...devices['iPhone 12'],
        isMobile: true,
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
  globalSetup: require.resolve('./tests/global-setup.ts'),
  globalTeardown: require.resolve('./tests/global-teardown.ts'),
  
  // Web server configuration
  webServer: {
    command: 'go run ./cmd/server',
    port: 8080,
    timeout: 120 * 1000, // 2 minutes
    reuseExistingServer: !process.env.CI,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
```

#### HTMX-Specific Test Utilities

**File**: `tests/utils/htmx-helpers.ts`

```typescript
import { Page, Locator, expect } from '@playwright/test';

export class HtmxTestHelper {
  constructor(private page: Page) {}

  /**
   * Wait for HTMX request to complete
   */
  async waitForHtmxRequest(triggerAction: () => Promise<void>, timeout = 5000) {
    const responsePromise = this.page.waitForEvent('response', {
      predicate: (response) => response.status() === 200 && 
                              response.headers()['hx-request'] !== undefined,
      timeout
    });
    
    await triggerAction();
    await responsePromise;
    
    // Wait for DOM to stabilize after HTMX update
    await this.page.waitForTimeout(100);
  }

  /**
   * Wait for HTMX indicator to appear and disappear
   */
  async waitForHtmxIndicator(timeout = 5000) {
    // Wait for indicator to appear
    await this.page.locator('.htmx-indicator').waitFor({ 
      state: 'visible', 
      timeout 
    });
    
    // Wait for indicator to disappear
    await this.page.locator('.htmx-indicator').waitFor({ 
      state: 'hidden', 
      timeout 
    });
  }

  /**
   * Test HTMX form submission
   */
  async submitHtmxForm(formSelector: string, expectedResponseSelector: string) {
    const form = this.page.locator(formSelector);
    await expect(form).toBeVisible();
    
    await this.waitForHtmxRequest(async () => {
      await form.locator('[type="submit"]').click();
    });
    
    // Verify response content appeared
    await expect(this.page.locator(expectedResponseSelector)).toBeVisible();
  }

  /**
   * Test dynamic content loading with HTMX
   */
  async testDynamicLoading(
    triggerSelector: string, 
    targetSelector: string, 
    expectedContent: string | RegExp
  ) {
    const trigger = this.page.locator(triggerSelector);
    const target = this.page.locator(targetSelector);
    
    await expect(trigger).toBeVisible();
    
    await this.waitForHtmxRequest(async () => {
      await trigger.click();
    });
    
    if (typeof expectedContent === 'string') {
      await expect(target).toContainText(expectedContent);
    } else {
      await expect(target).toHaveText(expectedContent);
    }
  }

  /**
   * Test HTMX pagination
   */
  async testPagination(nextButtonSelector: string, contentSelector: string) {
    const nextButton = this.page.locator(nextButtonSelector);
    const content = this.page.locator(contentSelector);
    
    // Capture initial content
    const initialContent = await content.textContent();
    
    await this.waitForHtmxRequest(async () => {
      await nextButton.click();
    });
    
    // Verify content changed
    const newContent = await content.textContent();
    expect(newContent).not.toBe(initialContent);
  }

  /**
   * Test HTMX search functionality
   */
  async testSearch(searchInputSelector: string, resultsSelector: string, searchTerm: string) {
    const searchInput = this.page.locator(searchInputSelector);
    const results = this.page.locator(resultsSelector);
    
    await expect(searchInput).toBeVisible();
    
    // Type search term and wait for HTMX request
    await this.waitForHtmxRequest(async () => {
      await searchInput.fill(searchTerm);
      // Trigger change event for HTMX
      await searchInput.press('Tab');
    });
    
    // Verify search results updated
    await expect(results).toBeVisible();
    await expect(results).toContainText(searchTerm, { ignoreCase: true });
  }

  /**
   * Test HTMX history functionality
   */
  async testHistoryManagement(linkSelector: string) {
    const currentUrl = this.page.url();
    const link = this.page.locator(linkSelector);
    
    await this.waitForHtmxRequest(async () => {
      await link.click();
    });
    
    // Verify URL changed (if using hx-push-url)
    const newUrl = this.page.url();
    
    // Go back
    await this.page.goBack();
    
    // Verify we're back to original state
    expect(this.page.url()).toBe(currentUrl);
  }

  /**
   * Verify HTMX attributes are properly set
   */
  async verifyHtmxAttributes(selector: string, expectedAttributes: Record<string, string>) {
    const element = this.page.locator(selector);
    
    for (const [attr, expectedValue] of Object.entries(expectedAttributes)) {
      const actualValue = await element.getAttribute(attr);
      expect(actualValue).toBe(expectedValue);
    }
  }

  /**
   * Test HTMX error handling
   */
  async testErrorHandling(triggerSelector: string, expectedErrorSelector: string) {
    // Mock a server error
    await this.page.route('**/api/**', (route) => {
      route.fulfill({ status: 500, body: 'Server Error' });
    });
    
    const trigger = this.page.locator(triggerSelector);
    await trigger.click();
    
    // Verify error message appears
    await expect(this.page.locator(expectedErrorSelector)).toBeVisible();
  }
}

// Additional utility functions for HTMX testing
export async function mockHtmxResponse(
  page: Page, 
  url: string, 
  responseBody: string, 
  status = 200
) {
  await page.route(url, (route) => {
    route.fulfill({
      status,
      headers: {
        'Content-Type': 'text/html',
        'HX-Request': 'true'
      },
      body: responseBody
    });
  });
}

export async function enableHtmxLogging(page: Page) {
  await page.addInitScript(() => {
    // Enable HTMX logging for debugging
    (window as any).htmx.logAll();
  });
}
```

#### Cross-Browser Test Suite

**File**: `tests/e2e/htmx-functionality.spec.ts`

```typescript
import { test, expect } from '@playwright/test';
import { HtmxTestHelper, enableHtmxLogging } from '../utils/htmx-helpers';

test.describe('HTMX Functionality', () => {
  let htmxHelper: HtmxTestHelper;

  test.beforeEach(async ({ page }) => {
    htmxHelper = new HtmxTestHelper(page);
    
    // Enable HTMX logging for debugging
    await enableHtmxLogging(page);
    
    // Navigate to the main page
    await page.goto('/');
    
    // Verify page loaded successfully
    await expect(page.locator('body')).toBeVisible();
  });

  test.describe('Dynamic Content Loading', () => {
    test('should load articles dynamically', async ({ page }) => {
      await htmxHelper.testDynamicLoading(
        '[data-testid="load-more-articles"]',
        '[data-testid="articles-container"]',
        /Article \d+/
      );
    });

    test('should handle infinite scroll', async ({ page }) => {
      const articlesContainer = page.locator('[data-testid="articles-container"]');
      
      // Scroll to trigger loading
      await page.evaluate(() => {
        window.scrollTo(0, document.body.scrollHeight);
      });
      
      // Wait for new content to load
      await htmxHelper.waitForHtmxRequest(async () => {
        // Trigger is automatic on scroll
      });
      
      // Verify new content appeared
      const articleCount = await articlesContainer.locator('.article').count();
      expect(articleCount).toBeGreaterThan(0);
    });

    test('should update content without page refresh', async ({ page }) => {
      const contentArea = page.locator('[data-testid="dynamic-content"]');
      const updateButton = page.locator('[data-testid="update-content"]');
      
      const initialContent = await contentArea.textContent();
      
      await htmxHelper.waitForHtmxRequest(async () => {
        await updateButton.click();
      });
      
      const updatedContent = await contentArea.textContent();
      expect(updatedContent).not.toBe(initialContent);
    });
  });

  test.describe('Search Functionality', () => {
    test('should perform live search', async ({ page }) => {
      await htmxHelper.testSearch(
        '[data-testid="search-input"]',
        '[data-testid="search-results"]',
        'test query'
      );
    });

    test('should clear search results', async ({ page }) => {
      const searchInput = page.locator('[data-testid="search-input"]');
      const searchResults = page.locator('[data-testid="search-results"]');
      
      // Perform search
      await htmxHelper.testSearch(
        '[data-testid="search-input"]',
        '[data-testid="search-results"]',
        'test'
      );
      
      // Clear search
      await htmxHelper.waitForHtmxRequest(async () => {
        await searchInput.clear();
        await searchInput.press('Tab');
      });
      
      // Verify results cleared
      await expect(searchResults).toBeEmpty();
    });

    test('should handle search with no results', async ({ page }) => {
      await htmxHelper.testSearch(
        '[data-testid="search-input"]',
        '[data-testid="search-results"]',
        'nonexistentquery12345'
      );
      
      await expect(page.locator('[data-testid="no-results"]')).toBeVisible();
    });
  });

  test.describe('Form Interactions', () => {
    test('should submit forms via HTMX', async ({ page }) => {
      await htmxHelper.submitHtmxForm(
        '[data-testid="article-form"]',
        '[data-testid="form-response"]'
      );
    });

    test('should validate form inputs', async ({ page }) => {
      const form = page.locator('[data-testid="article-form"]');
      const submitButton = form.locator('[type="submit"]');
      
      // Try to submit empty form
      await submitButton.click();
      
      // Check for validation messages
      await expect(page.locator('.error-message')).toBeVisible();
    });

    test('should handle form submission errors', async ({ page }) => {
      await htmxHelper.testErrorHandling(
        '[data-testid="error-form"] [type="submit"]',
        '[data-testid="error-message"]'
      );
    });
  });

  test.describe('Navigation and History', () => {
    test('should update URL with hx-push-url', async ({ page }) => {
      const navigationLink = page.locator('[data-testid="nav-link"]');
      const initialUrl = page.url();
      
      await htmxHelper.waitForHtmxRequest(async () => {
        await navigationLink.click();
      });
      
      const newUrl = page.url();
      expect(newUrl).not.toBe(initialUrl);
    });

    test('should handle browser back/forward', async ({ page }) => {
      await htmxHelper.testHistoryManagement('[data-testid="nav-link"]');
    });

    test('should preserve state during navigation', async ({ page }) => {
      const searchInput = page.locator('[data-testid="search-input"]');
      const navigationLink = page.locator('[data-testid="nav-link"]');
      
      // Set search term
      await searchInput.fill('test search');
      
      // Navigate away and back
      await htmxHelper.waitForHtmxRequest(async () => {
        await navigationLink.click();
      });
      
      await page.goBack();
      
      // Verify search term preserved
      await expect(searchInput).toHaveValue('test search');
    });
  });

  test.describe('Performance', () => {
    test('should load quickly', async ({ page }) => {
      const startTime = Date.now();
      await page.goto('/');
      const loadTime = Date.now() - startTime;
      
      expect(loadTime).toBeLessThan(3000); // 3 second limit
    });

    test('should handle rapid interactions', async ({ page }) => {
      const button = page.locator('[data-testid="rapid-click-button"]');
      
      // Click rapidly 5 times
      for (let i = 0; i < 5; i++) {
        await button.click();
        await page.waitForTimeout(100);
      }
      
      // Verify system didn't break
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
```

### Step 3.2: Accessibility Testing Implementation *(30 minutes)*

#### Accessibility Test Suite

**File**: `tests/e2e/accessibility.spec.ts`

```typescript
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Accessibility Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should not have any automatically detectable accessibility issues', async ({ page }) => {
    const accessibilityScanResults = await new AxeBuilder({ page }).analyze();
    
    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('should have proper ARIA attributes', async ({ page }) => {
    // Check main navigation
    const nav = page.locator('nav[role="navigation"]');
    await expect(nav).toBeVisible();
    
    // Check search input has proper labels
    const searchInput = page.locator('[data-testid="search-input"]');
    await expect(searchInput).toHaveAttribute('aria-label');
    
    // Check buttons have accessible names
    const buttons = page.locator('button');
    const buttonCount = await buttons.count();
    
    for (let i = 0; i < buttonCount; i++) {
      const button = buttons.nth(i);
      const hasAriaLabel = await button.getAttribute('aria-label');
      const hasInnerText = await button.textContent();
      
      expect(hasAriaLabel || hasInnerText).toBeTruthy();
    }
  });

  test('should be keyboard navigable', async ({ page }) => {
    // Start from search input
    const searchInput = page.locator('[data-testid="search-input"]');
    await searchInput.focus();
    
    // Tab through interactive elements
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    
    // Verify focus moved to expected element
    const focusedElement = page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });

  test('should have sufficient color contrast', async ({ page }) => {
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    const colorContrastViolations = accessibilityScanResults.violations.filter(
      violation => violation.id === 'color-contrast'
    );
    
    expect(colorContrastViolations).toEqual([]);
  });

  test('should work with screen readers', async ({ page }) => {
    // Check for proper heading structure
    const h1 = page.locator('h1');
    await expect(h1).toBeVisible();
    
    // Check for landmark regions
    const main = page.locator('main');
    await expect(main).toBeVisible();
    
    // Check for skip links
    const skipLink = page.locator('a[href="#main-content"]');
    if (await skipLink.count() > 0) {
      await expect(skipLink).toHaveText(/skip to main content/i);
    }
  });
});
```

### Step 3.3: Mobile Responsiveness Testing *(15 minutes)*

#### Mobile-Specific Test Suite

**File**: `tests/e2e/mobile-responsiveness.spec.ts`

```typescript
import { test, expect, devices } from '@playwright/test';

// Test on mobile devices
for (const deviceName of ['iPhone 12', 'Pixel 5', 'iPad']) {
  test.describe(`Mobile Tests - ${deviceName}`, () => {
    test.use(devices[deviceName]);

    test('should display properly on mobile', async ({ page }) => {
      await page.goto('/');
      
      // Check viewport is mobile-sized
      const viewport = page.viewportSize();
      expect(viewport!.width).toBeLessThanOrEqual(768);
      
      // Verify mobile navigation
      const mobileNav = page.locator('[data-testid="mobile-nav"]');
      if (await mobileNav.count() > 0) {
        await expect(mobileNav).toBeVisible();
      }
      
      // Check responsive layout
      const mainContent = page.locator('main');
      await expect(mainContent).toBeVisible();
    });

    test('should handle touch interactions', async ({ page }) => {
      await page.goto('/');
      
      const touchButton = page.locator('[data-testid="touch-button"]');
      if (await touchButton.count() > 0) {
        await touchButton.tap();
        await expect(page.locator('[data-testid="touch-response"]')).toBeVisible();
      }
    });

    test('should scroll properly on mobile', async ({ page }) => {
      await page.goto('/');
      
      // Scroll down
      await page.evaluate(() => {
        window.scrollTo(0, window.innerHeight);
      });
      
      // Verify scroll position changed
      const scrollY = await page.evaluate(() => window.scrollY);
      expect(scrollY).toBeGreaterThan(0);
    });

    test('should handle orientation changes', async ({ page, context }) => {
      await page.goto('/');
      
      // Change to landscape
      await page.setViewportSize({ width: 896, height: 414 });
      
      // Verify layout adapts
      const content = page.locator('main');
      await expect(content).toBeVisible();
      
      // Change back to portrait
      await page.setViewportSize({ width: 414, height: 896 });
      
      // Verify layout still works
      await expect(content).toBeVisible();
    });
  });
}
```

### Step 3.4: Performance Testing *(30 minutes)*

#### Performance Validation Suite

**File**: `tests/e2e/performance.spec.ts`

```typescript
import { test, expect } from '@playwright/test';

test.describe('Performance Tests', () => {
  test('should meet Core Web Vitals thresholds', async ({ page }) => {
    await page.goto('/');
    
    // Measure Core Web Vitals
    const webVitals = await page.evaluate(() => {
      return new Promise((resolve) => {
        const vitals: any = {};
        
        // Largest Contentful Paint
        new PerformanceObserver((list) => {
          const entries = list.getEntries();
          vitals.lcp = entries[entries.length - 1]?.startTime;
        }).observe({ entryTypes: ['largest-contentful-paint'] });
        
        // First Input Delay (simulated)
        vitals.fid = 0; // Would be measured on real user interaction
        
        // Cumulative Layout Shift
        let clsValue = 0;
        new PerformanceObserver((list) => {
          for (const entry of list.getEntries() as any[]) {
            if (!entry.hadRecentInput) {
              clsValue += entry.value;
            }
          }
          vitals.cls = clsValue;
        }).observe({ entryTypes: ['layout-shift'] });
        
        setTimeout(() => resolve(vitals), 3000);
      });
    });
    
    // Verify Core Web Vitals thresholds
    expect(webVitals.lcp).toBeLessThan(2500); // 2.5s
    expect(webVitals.fid).toBeLessThan(100);  // 100ms
    expect(webVitals.cls).toBeLessThan(0.1);  // 0.1
  });

  test('should load resources efficiently', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/');
    
    const loadTime = Date.now() - startTime;
    expect(loadTime).toBeLessThan(3000); // 3 second page load
    
    // Check resource sizes
    const resources = await page.evaluate(() => {
      return performance.getEntriesByType('resource').map(r => ({
        name: r.name,
        size: (r as any).transferSize || 0,
        duration: r.duration
      }));
    });
    
    // Verify no excessively large resources
    const largeResources = resources.filter(r => r.size > 1024 * 1024); // 1MB
    expect(largeResources).toHaveLength(0);
  });

  test('should handle concurrent users simulation', async ({ page, context }) => {
    // Simulate multiple users by opening multiple pages
    const pages = await Promise.all([
      context.newPage(),
      context.newPage(),
      context.newPage(),
    ]);
    
    pages.push(page); // Include original page
    
    // Navigate all pages simultaneously
    await Promise.all(pages.map(p => p.goto('/')));
    
    // Perform actions on all pages simultaneously
    await Promise.all(pages.map(async (p) => {
      const searchInput = p.locator('[data-testid="search-input"]');
      if (await searchInput.count() > 0) {
        await searchInput.fill('test');
      }
    }));
    
    // Verify all pages still functional
    for (const p of pages) {
      await expect(p.locator('body')).toBeVisible();
    }
    
    // Close additional pages
    await Promise.all(pages.slice(0, 3).map(p => p.close()));
  });

  test('should handle memory efficiently', async ({ page }) => {
    await page.goto('/');
    
    // Perform memory-intensive operations
    for (let i = 0; i < 10; i++) {
      const loadButton = page.locator('[data-testid="load-more-articles"]');
      if (await loadButton.count() > 0) {
        await loadButton.click();
        await page.waitForTimeout(500);
      }
    }
    
    // Check for memory leaks (basic check)
    const memoryUsage = await page.evaluate(() => {
      if ('memory' in performance) {
        return (performance as any).memory.usedJSHeapSize;
      }
      return 0;
    });
    
    // Verify memory usage is reasonable (< 50MB)
    if (memoryUsage > 0) {
      expect(memoryUsage).toBeLessThan(50 * 1024 * 1024);
    }
  });
});
```

## 📚 Enhanced Playwright Best Practices

### Web-First Assertions for HTMX Applications

Based on Playwright's documentation, use web-first assertions that automatically wait for conditions:

```typescript
// ✅ Recommended: Web-first assertions
await expect(page.getByText('Article loaded')).toBeVisible();
await expect(page.locator('.htmx-indicator')).toBeHidden();

// ❌ Avoid: Manual checks
expect(await page.getByText('Article loaded').isVisible()).toBe(true);
```

### Test Isolation and Browser Context

Playwright ensures test isolation by providing fresh browser contexts:

```typescript
test('isolated test 1', async ({ page }) => {
  // Fresh browser context for this test
  await page.goto('/');
});

test('isolated test 2', async ({ page }) => {
  // Completely isolated from test 1
  await page.goto('/');
});
```

### Network Mocking for HTMX Applications

Mock server responses for consistent testing:

```typescript
test('should handle API errors gracefully', async ({ page }) => {
  // Mock server error
  await page.route('**/api/articles', route => route.fulfill({
    status: 500,
    body: JSON.stringify({ error: 'Server Error' })
  }));
  
  await page.goto('/');
  
  // Verify error handling
  await expect(page.locator('.error-message')).toBeVisible();
});
```

## 🎯 Acceptance Criteria

### Phase 3 Completion Checklist
- [ ] **E2E Test Pass Rate**: >85% across all browsers
- [ ] **HTMX Functionality**: All dynamic content loading works reliably
- [ ] **Cross-Browser Compatibility**: Consistent behavior on Chromium, Firefox, WebKit
- [ ] **Mobile Responsiveness**: All tests pass on mobile devices
- [ ] **Accessibility Compliance**: WCAG 2.1 AA standards met
- [ ] **Performance Standards**: Core Web Vitals thresholds achieved
- [ ] **Error Handling**: Graceful degradation when server errors occur
- [ ] **Test Reliability**: Minimal flaky tests (<5% failure rate)

### Quality Gates
1. **HTMX Integration Gate**: All HTMX interactions work without page refresh
2. **Cross-Browser Gate**: <5% difference in test results between browsers
3. **Performance Gate**: Page load <3s, LCP <2.5s, CLS <0.1
4. **Accessibility Gate**: Zero axe-core violations for WCAG 2.1 AA

## 🔍 Validation Commands

### Test Phase 3 Success
Run these commands to validate Phase 3 completion:

```powershell
# 1. Run cross-browser E2E tests
npx playwright test --project=chromium,firefox,webkit
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Cross-Browser Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Cross-Browser Tests: FAIL" -ForegroundColor Red }

# 2. Run mobile tests
npx playwright test --project="Mobile Chrome","Mobile Safari"
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Mobile Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Mobile Tests: FAIL" -ForegroundColor Red }

# 3. Run accessibility tests
npx playwright test accessibility.spec.ts
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Accessibility Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Accessibility Tests: FAIL" -ForegroundColor Red }

# 4. Run performance tests
npx playwright test performance.spec.ts
if ($LASTEXITCODE -eq 0) { Write-Host "✅ Performance Tests: PASS" -ForegroundColor Green } else { Write-Host "❌ Performance Tests: FAIL" -ForegroundColor Red }

# 5. Generate comprehensive report
npx playwright show-report
```

## 🚫 Rollback Procedures

### If E2E Tests Continue to Fail
```powershell
# Rollback script: rollback-phase3-e2e.ps1
Write-Host "🔄 Rolling back Phase 3 E2E changes..." -ForegroundColor Yellow

# Stop any running Playwright processes
Get-Process | Where-Object { $_.ProcessName -like "*playwright*" } | Stop-Process -Force -ErrorAction SilentlyContinue

# Clean up test artifacts
$cleanupDirs = @("test-results", "playwright-report", "test-results/videos", "test-results/traces")
foreach ($dir in $cleanupDirs) {
    if (Test-Path $dir) {
        Remove-Item $dir -Recurse -Force
        Write-Host "  ✅ Cleaned directory: $dir" -ForegroundColor Green
    }
}

# Restore original playwright config if backup exists
if (Test-Path "playwright.config.ts.backup") {
    Move-Item "playwright.config.ts.backup" "playwright.config.ts" -Force
    Write-Host "  ✅ Restored Playwright configuration" -ForegroundColor Green
}

Write-Host "🔄 Phase 3 rollback complete" -ForegroundColor Green
```

## 💡 HTMX Testing Best Practices

### Dynamic Content Loading Patterns
```typescript
// Pattern 1: Test infinite scroll
test('infinite scroll loading', async ({ page }) => {
  await page.goto('/articles');
  
  // Initial article count
  const initialCount = await page.locator('.article').count();
  
  // Scroll to trigger loading
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  
  // Wait for HTMX request
  await page.waitForResponse('**/api/articles*');
  
  // Verify new articles loaded
  const newCount = await page.locator('.article').count();
  expect(newCount).toBeGreaterThan(initialCount);
});
```

### Form Interaction Testing
```typescript
// Pattern 2: Test form submission with HTMX
test('form submission updates content', async ({ page }) => {
  await page.goto('/articles/new');
  
  await page.fill('[name="title"]', 'Test Article');
  await page.fill('[name="content"]', 'Test Content');
  
  // Submit form and wait for HTMX response
  const responsePromise = page.waitForResponse('**/api/articles');
  await page.click('[type="submit"]');
  await responsePromise;
  
  // Verify success message
  await expect(page.locator('.success-message')).toBeVisible();
});
```

### Search Functionality Testing
```typescript
// Pattern 3: Test live search
test('live search updates results', async ({ page }) => {
  await page.goto('/');
  
  const searchInput = page.locator('[data-testid="search-input"]');
  const results = page.locator('[data-testid="search-results"]');
  
  // Type search term
  await searchInput.type('test');
  
  // Wait for debounced HTMX request
  await page.waitForResponse('**/api/search*');
  
  // Verify results updated
  await expect(results).toContainText('test');
});
```

## 📞 Support & Escalation

**If you encounter issues during Phase 3:**

1. **E2E Test Failures**: Check server is running, verify network connectivity
2. **Cross-Browser Issues**: Check for browser-specific CSS/JS compatibility
3. **HTMX Problems**: Verify HTMX library loaded, check network requests in DevTools
4. **Performance Issues**: Profile with Chrome DevTools, optimize images/scripts
5. **Accessibility Failures**: Use axe DevTools for detailed violation analysis

**Common Solutions**:
- **Flaky Tests**: Add proper waits, use `page.waitForResponse()` for HTMX requests
- **Timeout Errors**: Increase timeouts in playwright.config.ts
- **Mobile Issues**: Test viewport settings, touch events configuration
- **Memory Leaks**: Check for event listener cleanup, large DOM structures

---

**Phase 3 Status**: ⏳ READY FOR IMPLEMENTATION  
**Next Phase**: `05_phase4_process_integration.md`  
**Estimated Total Time**: 1-2 hours  
**Key Dependencies**: Node.js 18+, Playwright installed, Server running on port 8080
