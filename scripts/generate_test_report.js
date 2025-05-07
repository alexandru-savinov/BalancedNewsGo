const fs = require('fs');
const path = require('path');

// Function to read test results
function readTestResults(filePath) {
  try {
    const data = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(data);
  } catch (err) {
    console.error(`Error reading file ${filePath}: ${err}`);
    return null;
  }
}

// Function to generate HTML report
function generateHtmlReport(results) {
  let html = `
<!DOCTYPE html>
<html>
<head>
  <title>Postman Test Results</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; }
    h1, h2, h3 { color: #333; }
    .test-group { margin-bottom: 20px; border: 1px solid #ddd; padding: 10px; border-radius: 5px; }
    .test-item { margin-bottom: 10px; padding: 10px; border-radius: 5px; }
    .pass { background-color: #dff0d8; }
    .fail { background-color: #f2dede; }
    .summary { font-weight: bold; margin-bottom: 10px; }
    .details { margin-left: 20px; }
    pre { background-color: #f5f5f5; padding: 10px; border-radius: 5px; overflow-x: auto; }
  </style>
</head>
<body>
  <h1>Postman Test Results</h1>
`;

  // Add summary
  let totalTests = 0;
  let passedTests = 0;

  for (const result of results) {
    if (result && result.run && result.run.stats) {
      totalTests += result.run.stats.assertions.total;
      passedTests += result.run.stats.assertions.total - result.run.stats.assertions.failed;
    }
  }

  html += `
  <div class="summary">
    <p>Total Tests: ${totalTests}</p>
    <p>Passed Tests: ${passedTests}</p>
    <p>Failed Tests: ${totalTests - passedTests}</p>
    <p>Pass Rate: ${((passedTests / totalTests) * 100).toFixed(2)}%</p>
  </div>
`;

  // Add details for each test file
  for (const result of results) {
    if (!result || !result.collection || !result.run) continue;

    html += `
  <div class="test-group">
    <h2>${result.collection.info.name}</h2>
`;

    // Add execution summary
    if (result.run.stats) {
      const stats = result.run.stats;
      html += `
    <div class="summary">
      <p>Iterations: ${stats.iterations.total} (Failed: ${stats.iterations.failed})</p>
      <p>Requests: ${stats.requests.total} (Failed: ${stats.requests.failed})</p>
      <p>Test Scripts: ${stats.testScripts.total} (Failed: ${stats.testScripts.failed})</p>
      <p>Assertions: ${stats.assertions.total} (Failed: ${stats.assertions.failed})</p>
      <p>Total Run Duration: ${result.run.timings.completed - result.run.timings.started}ms</p>
    </div>
`;
    }

    // Add execution details
    if (result.run.executions) {
      for (const execution of result.run.executions) {
        const item = execution.item;
        const response = execution.response;
        const assertions = execution.assertions || [];

        const hasFailedAssertions = assertions.some(a => !a.passed);
        const statusClass = hasFailedAssertions ? 'fail' : 'pass';

        html += `
    <div class="test-item ${statusClass}">
      <h3>${item.name}</h3>
      <div class="details">
        <p><strong>Request:</strong> ${execution.request.method} ${execution.request.url.toString()}</p>
        <p><strong>Status:</strong> ${response ? response.code : 'N/A'} ${response ? response.status : ''}</p>
        <p><strong>Response Time:</strong> ${response ? response.responseTime : 'N/A'}ms</p>
        <p><strong>Assertions:</strong></p>
        <ul>
`;

        for (const assertion of assertions) {
          const assertionClass = assertion.passed ? 'pass' : 'fail';
          html += `          <li class="${assertionClass}">${assertion.assertion}: ${assertion.passed ? 'Passed' : 'Failed'}</li>\n`;
          if (!assertion.passed && assertion.error) {
            html += `          <li class="fail">Error: ${assertion.error.message}</li>\n`;
          }
        }

        html += `
        </ul>
`;

        if (response && response.body) {
          html += `
        <p><strong>Response Body:</strong></p>
        <pre>${JSON.stringify(JSON.parse(response.body), null, 2)}</pre>
`;
        }

        html += `
      </div>
    </div>
`;
      }
    }

    html += `
  </div>
`;
  }

  html += `
</body>
</html>
`;

  return html;
}

// Main function
function main() {
  const testResultsDir = path.join(__dirname, 'test-results');
  const outputFile = path.join(testResultsDir, 'test_report.html');

  // Get all JSON files in the test-results directory
  const files = fs.readdirSync(testResultsDir)
    .filter(file => file.endsWith('.json'))
    .map(file => path.join(testResultsDir, file));

  // Read test results
  const results = files.map(file => readTestResults(file)).filter(result => result !== null);

  // Generate HTML report
  const html = generateHtmlReport(results);

  // Write HTML report to file
  fs.writeFileSync(outputFile, html);

  console.log(`Test report generated: ${outputFile}`);
}

main();