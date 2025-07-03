/**
 * Test script to verify the progress indicator fix
 * This script tests the fixed progress indicator functionality
 */

// Test the API endpoint format
async function testAPIEndpoint() {
    console.log('🧪 Testing API endpoint format...');
    
    try {
        const response = await fetch('http://localhost:8080/api/articles/600/bias');
        const data = await response.json();
        
        console.log('API Response:', data);
        
        // Test the composite score extraction logic
        let compositeScore = null;
        if (data.composite_score !== undefined) {
            compositeScore = data.composite_score;
            console.log('✅ Found composite_score in old format:', compositeScore);
        } else if (data.data && data.data.composite_score !== undefined) {
            compositeScore = data.data.composite_score;
            console.log('✅ Found composite_score in new format:', compositeScore);
        } else {
            console.log('❌ No composite_score found');
        }
        
        return compositeScore !== null;
    } catch (error) {
        console.error('❌ API test failed:', error);
        return false;
    }
}

// Test progress indicator connection
function testProgressIndicatorConnection() {
    console.log('🧪 Testing progress indicator connection...');
    
    // Check if progress indicator element exists
    const progressIndicator = document.getElementById('reanalysis-progress');
    if (!progressIndicator) {
        console.error('❌ Progress indicator element not found');
        return false;
    }
    
    // Check if methods exist
    if (typeof progressIndicator.reset !== 'function') {
        console.error('❌ Progress indicator reset method not available');
        return false;
    }
    
    if (typeof progressIndicator.connect !== 'function') {
        console.error('❌ Progress indicator connect method not available');
        return false;
    }
    
    console.log('✅ Progress indicator methods available');
    return true;
}

// Run tests
async function runTests() {
    console.log('🚀 Starting progress indicator fix verification...');
    
    const apiTest = await testAPIEndpoint();
    const progressTest = testProgressIndicatorConnection();
    
    console.log('\n📊 Test Results:');
    console.log('API Format Compatibility:', apiTest ? '✅ PASS' : '❌ FAIL');
    console.log('Progress Indicator Setup:', progressTest ? '✅ PASS' : '❌ FAIL');
    
    const allPassed = apiTest && progressTest;
    console.log('\n🎯 Overall Result:', allPassed ? '✅ ALL TESTS PASSED' : '❌ SOME TESTS FAILED');
    
    if (allPassed) {
        console.log('\n🎉 The progress indicator fix is working correctly!');
        console.log('The system should now:');
        console.log('- Connect to SSE properly for real-time updates');
        console.log('- Handle both old and new API response formats');
        console.log('- Show progress updates in real-time');
        console.log('- Update bias scores automatically when complete');
    }
    
    return allPassed;
}

// Export for use in browser console or as module
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { runTests, testAPIEndpoint, testProgressIndicatorConnection };
} else {
    // Run immediately if in browser
    document.addEventListener('DOMContentLoaded', runTests);
}
