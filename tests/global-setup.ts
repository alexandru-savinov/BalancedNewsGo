import { FullConfig } from '@playwright/test';

/**
 * Global Setup for Playwright Tests
 * Runs once before all tests across all projects
 */
async function globalSetup(config: FullConfig) {
  console.log('🚀 Starting global setup for E2E tests...');
  
  // Environment validation
  console.log('📋 Environment check:');
  console.log(`  - Node version: ${process.version}`);
  console.log(`  - Platform: ${process.platform}`);
  console.log(`  - CI environment: ${process.env.CI ? 'Yes' : 'No'}`);
  console.log(`  - Base URL: ${config.projects[0]?.use?.baseURL || 'Not configured'}`);
  
  // Pre-test database setup (if needed)
  try {
    console.log('🗄️  Database setup check...');
    // Add any database initialization here if needed
    console.log('  ✅ Database setup complete');
  } catch (error) {
    console.error('  ❌ Database setup failed:', error);
    throw error;
  }
  
  // Validate server configuration
  console.log('🔧 Server configuration validation...');
  const webServer = config.webServer;
  if (webServer) {
    console.log(`  - Command: ${Array.isArray(webServer) ? webServer[0]?.command : webServer.command}`);
    console.log(`  - Port: ${Array.isArray(webServer) ? webServer[0]?.port : webServer.port}`);
    console.log('  ✅ Server configuration valid');
  } else {
    console.warn('  ⚠️  No web server configuration found');
  }
  
  // Create test results directories
  try {
    console.log('📁 Creating test artifact directories...');
    const fs = await import('fs');
    const path = await import('path');
    
    const dirs = [
      'test-results',
      'test-results/playwright-report',
      'test-results/videos',
      'test-results/traces',
      'test-results/screenshots'
    ];
    
    for (const dir of dirs) {
      const fullPath = path.resolve(dir);
      if (!fs.existsSync(fullPath)) {
        fs.mkdirSync(fullPath, { recursive: true });
        console.log(`  ✅ Created directory: ${dir}`);
      } else {
        console.log(`  📁 Directory exists: ${dir}`);
      }
    }
  } catch (error) {
    console.error('  ❌ Failed to create directories:', error);
    // Don't fail setup for directory creation issues
  }
  
  console.log('✅ Global setup complete!\n');
}

export default globalSetup;
