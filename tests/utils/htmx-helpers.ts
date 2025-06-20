import { Page, expect } from '@playwright/test';

/**
 * HTMX Test Helper Utilities
 * Enhanced utilities for testing HTMX applications with Playwright
 */
export class HtmxTestHelper {
  constructor(private page: Page) {}
  /**
   * Wait for HTMX request to complete
   */
  async waitForHtmxRequest(triggerAction: () => Promise<void>, timeout = 5000) {
    const requestPromise = this.page.waitForEvent('request', {
      predicate: (request) => request.headers()['hx-request'] !== undefined ||
                              request.headers()['HX-Request'] !== undefined ||
                              request.url().includes('/htmx/'),
      timeout
    });
    
    const responsePromise = this.page.waitForEvent('response', {
      predicate: (response) => response.status() === 200 && 
                              (response.url().includes('/htmx/') || 
                               response.request().headers()['hx-request'] !== undefined ||
                               response.request().headers()['HX-Request'] !== undefined),
      timeout
    });
    
    await triggerAction();
    await Promise.race([requestPromise, responsePromise]);
    
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
   * Test HTMX history functionality
   */
  async testHistoryManagement(linkSelector: string) {
    const currentUrl = this.page.url();
    const link = this.page.locator(linkSelector);
    
    await this.waitForHtmxRequest(async () => {
      await link.click();
    });
    
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
