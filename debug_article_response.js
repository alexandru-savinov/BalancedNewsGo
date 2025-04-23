// This script will create an article and log the full response
const http = require('http');
const fs = require('fs');

// Create a test article
const articleData = JSON.stringify({
  title: "Debug Article",
  content: "This is a test article for debugging.",
  url: "https://example.com/debug-" + Date.now(),
  pub_date: new Date().toISOString(),
  source: "test"
});

console.log("Sending request with data:", articleData);

const options = {
  hostname: 'localhost',
  port: 8080,
  path: '/api/articles',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': articleData.length
  }
};

const req = http.request(options, (res) => {
  console.log(`STATUS: ${res.statusCode}`);
  console.log(`HEADERS: ${JSON.stringify(res.headers)}`);
  
  let data = '';
  
  res.on('data', (chunk) => {
    data += chunk;
  });
  
  res.on('end', () => {
    console.log('Raw response data:');
    console.log(data);
    
    try {
      const jsonData = JSON.parse(data);
      console.log('Parsed JSON:');
      console.log(JSON.stringify(jsonData, null, 2));
      
      // Log all properties
      console.log('All properties:');
      for (const prop in jsonData) {
        console.log(`${prop}: ${jsonData[prop]}`);
      }
      
      // Save the response to a file for inspection
      fs.writeFileSync('article_response.json', JSON.stringify(jsonData, null, 2));
      console.log('Response saved to article_response.json');
    } catch (e) {
      console.error('Error parsing JSON:', e);
    }
  });
});

req.on('error', (e) => {
  console.error(`Problem with request: ${e.message}`);
});

req.write(articleData);
req.end();