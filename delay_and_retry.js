// Helper script to add a small delay and retry mechanism for article operations
const fetch = require('node-fetch');

/**
 * Delays execution by the specified number of milliseconds
 * @param {number} ms - Milliseconds to delay
 * @returns {Promise} A promise that resolves after the delay
 */
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Retries a fetch operation with delay between attempts
 * @param {string} url - URL to fetch
 * @param {Object} options - Fetch options
 * @param {number} maxRetries - Maximum number of retry attempts
 * @param {number} delayMs - Delay in milliseconds between retries
 * @returns {Promise} A promise that resolves with the fetch response
 */
async function fetchWithRetry(url, options = {}, maxRetries = 3, delayMs = 300) {
  let lastError;
  
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      // Wait before trying (except first attempt)
      if (attempt > 0) {
        console.log(`Retry attempt ${attempt} after delay of ${delayMs}ms...`);
        await delay(delayMs);
      }
      
      const response = await fetch(url, options);
      
      // If status is 5xx, consider it retriable
      if (response.status >= 500) {
        lastError = new Error(`Server error: ${response.status} ${response.statusText}`);
        console.log(`Received ${response.status} response, will retry...`);
        continue;
      }
      
      return response;
    } catch (err) {
      lastError = err;
      console.log(`Fetch error: ${err.message}, will retry...`);
    }
  }
  
  throw lastError || new Error(`Failed after ${maxRetries} attempts`);
}

module.exports = {
  delay,
  fetchWithRetry
};