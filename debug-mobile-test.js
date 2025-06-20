const { test, expect, devices } = require('@playwright/test');

console.log('Devices object:', devices ? 'exists' : 'null/undefined');
console.log('iPhone 12 device:', devices['iPhone 12'] ? 'exists' : 'null/undefined');

// Try the same approach as the test file
const mobileIPhoneTest = test.extend({
  page: async ({ browser }, use) => {
    console.log('Creating context with device...');
    const context = await browser.newContext(devices['iPhone 12']);
    const page = await context.newPage();
    await use(page);
    await context.close();
  },
});

console.log('Mobile test extended successfully');
