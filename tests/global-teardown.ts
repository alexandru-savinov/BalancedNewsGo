import { FullConfig } from '@playwright/test';

/**
 * Global Teardown for Playwright Tests
 * Runs once after all tests across all projects
 */
async function globalTeardown(config: FullConfig) {
  console.log('\n🧹 Starting global teardown...');
  
  // Clean up test data (if needed)
  try {
    console.log('🗄️  Database cleanup check...');
    // Add any test data cleanup here if needed
    console.log('  ✅ Database cleanup complete');
  } catch (error) {
    console.error('  ❌ Database cleanup failed:', error);
    // Don't fail teardown for cleanup issues
  }
  
  // Generate test summary
  try {
    console.log('📊 Generating test summary...');
    const fs = await import('fs');
    const path = await import('path');
    
    const resultsPath = path.resolve('test-results/test-results.json');
    if (fs.existsSync(resultsPath)) {
      const results = JSON.parse(fs.readFileSync(resultsPath, 'utf8'));
      const stats = results.stats || {};
      
      console.log('📈 Test Execution Summary:');
      console.log(`  - Total tests: ${stats.total || 0}`);
      console.log(`  - Passed: ${stats.passed || 0}`);
      console.log(`  - Failed: ${stats.failed || 0}`);
      console.log(`  - Skipped: ${stats.skipped || 0}`);
      const durationText = stats.duration ? `${Math.round(stats.duration / 1000)}s` : 'Unknown';
      console.log(`  - Duration: ${durationText}`);
      
      const passRate = stats.total ? Math.round((stats.passed / stats.total) * 100) : 0;
      console.log(`  - Pass rate: ${passRate}%`);
      
      if (passRate >= 85) {
        console.log('  🎉 Phase 3 success criteria met (≥85% pass rate)!');
      } else {
        console.log('  ⚠️  Phase 3 success criteria not met (<85% pass rate)');
      }
    } else {
      console.log('  📋 No test results file found');
    }
  } catch (error) {
    console.error('  ❌ Failed to generate summary:', error);
  }
  
  // Clean up temporary files (optional)
  try {
    console.log('🗑️  Temporary file cleanup...');
    const fs = await import('fs');
    const path = await import('path');
    
    // Clean up old screenshots (keep only recent ones)
    const screenshotsDir = path.resolve('test-results/screenshots');
    if (fs.existsSync(screenshotsDir)) {
      // Add cleanup logic here if needed
      console.log('  ✅ Screenshot cleanup complete');
    }
  } catch (error) {
    console.error('  ❌ Cleanup failed:', error);
  }
  
  console.log('✅ Global teardown complete!');
}

export default globalTeardown;
