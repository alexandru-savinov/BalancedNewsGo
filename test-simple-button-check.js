// Simple button responsiveness test
// Copy and paste this into browser console at http://localhost:8080/articles/584

console.log('🔍 SIMPLE BUTTON TEST');
console.log('====================');

// Check if elements exist
const reanalyzeBtn = document.getElementById('reanalyze-btn');
const btnText = document.getElementById('btn-text');
const btnLoading = document.getElementById('btn-loading');
const progressIndicator = document.getElementById('reanalysis-progress');

console.log('📋 Element Check:');
console.log('✅ Button:', reanalyzeBtn ? 'Found' : '❌ NOT FOUND');
console.log('✅ Button Text:', btnText ? 'Found' : '❌ NOT FOUND');
console.log('✅ Button Loading:', btnLoading ? 'Found' : '❌ NOT FOUND');
console.log('✅ Progress Indicator:', progressIndicator ? 'Found' : '❌ NOT FOUND');

if (reanalyzeBtn) {
    console.log('');
    console.log('📋 Button Properties:');
    console.log('   Article ID:', reanalyzeBtn.getAttribute('data-article-id'));
    console.log('   Disabled:', reanalyzeBtn.disabled);
    console.log('   Text Content:', btnText ? btnText.textContent : 'N/A');
    
    console.log('');
    console.log('🖱️ Testing Button Click...');
    
    // Add a test click listener
    const testClick = () => {
        console.log('✅ BUTTON CLICK DETECTED!');
        console.log('   The button is responsive.');
    };
    
    reanalyzeBtn.addEventListener('click', testClick, { once: true });
    
    console.log('👆 Now click the "Request Reanalysis" button on the page.');
    console.log('   If you see "BUTTON CLICK DETECTED!" message, the button is working.');
    console.log('   If not, there may be a JavaScript error preventing the click handler.');
    
} else {
    console.log('❌ Cannot test button - element not found!');
    console.log('💡 Check if you are on the correct page: http://localhost:8080/articles/584');
}

// Check for JavaScript errors
console.log('');
console.log('🔍 Checking for JavaScript errors...');
console.log('   (Check the Console tab for any red error messages)');

// Test ProgressIndicator methods
if (progressIndicator) {
    console.log('');
    console.log('📋 ProgressIndicator Methods:');
    const methods = ['reset', 'connect', 'disconnect', 'updateProgress'];
    methods.forEach(method => {
        const available = typeof progressIndicator[method] === 'function';
        console.log(`   ${available ? '✅' : '❌'} ${method}: ${available ? 'Available' : 'Missing'}`);
    });
}

console.log('');
console.log('🎯 NEXT STEPS:');
console.log('1. Click the "Request Reanalysis" button');
console.log('2. Check console for "BUTTON CLICK DETECTED!" message');
console.log('3. Look for any red error messages in console');
console.log('4. If button works, it should show "Processing..." briefly');
