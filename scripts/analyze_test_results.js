const fs = require('fs');
const path = require('path');
const readline = require('readline');

// ANSI color codes for terminal output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  underscore: '\x1b[4m',
  blink: '\x1b[5m',
  reverse: '\x1b[7m',
  hidden: '\x1b[8m',

  black: '\x1b[30m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',

  bgBlack: '\x1b[40m',
  bgRed: '\x1b[41m',
  bgGreen: '\x1b[42m',
  bgYellow: '\x1b[43m',
  bgBlue: '\x1b[44m',
  bgMagenta: '\x1b[45m',
  bgCyan: '\x1b[46m',
  bgWhite: '\x1b[47m'
};

// Function to read test results
function readTestResults(filePath) {
  try {
    const data = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(data);
  } catch (err) {
    console.error(`${colors.red}Error reading file ${filePath}: ${err}${colors.reset}`);
    return null;
  }
}

// Function to list all test result files
function listTestResultFiles() {
  const testResultsDir = path.join(__dirname, 'test-results');
  try {
    const files = fs.readdirSync(testResultsDir)
      .filter(file => file.endsWith('.json'))
      .map(file => path.join(testResultsDir, file));
    return files;
  } catch (err) {
    console.error(`${colors.red}Error reading test-results directory: ${err}${colors.reset}`);
    return [];
  }
}

// Function to print test summary
function printTestSummary(result) {
  if (!result || !result.collection || !result.run) {
    console.log(`${colors.red}Invalid test result file${colors.reset}`);
    return;
  }

  console.log(`\n${colors.bright}${colors.cyan}Test Collection: ${result.collection.info.name}${colors.reset}`);

  if (result.run.stats) {
    const stats = result.run.stats;
    console.log(`\n${colors.bright}Execution Summary:${colors.reset}`);
    console.log(`  Iterations: ${stats.iterations.total} (Failed: ${stats.iterations.failed})`);
    console.log(`  Requests: ${stats.requests.total} (Failed: ${stats.requests.failed})`);
    console.log(`  Test Scripts: ${stats.testScripts.total} (Failed: ${stats.testScripts.failed})`);
    console.log(`  Assertions: ${stats.assertions.total} (Failed: ${stats.assertions.failed})`);
    console.log(`  Total Run Duration: ${result.run.timings.completed - result.run.timings.started}ms`);
  }

  if (result.run.failures && result.run.failures.length > 0) {
    console.log(`\n${colors.bright}${colors.red}Failures:${colors.reset}`);
    result.run.failures.forEach((failure, index) => {
      console.log(`  ${index + 1}. ${colors.red}${failure.error.message}${colors.reset}`);
      console.log(`     at ${failure.source.name}`);
    });
  }
}

// Function to print detailed test results
function printDetailedTestResults(result) {
  if (!result || !result.collection || !result.run) {
    console.log(`${colors.red}Invalid test result file${colors.reset}`);
    return;
  }

  console.log(`\n${colors.bright}${colors.cyan}Detailed Test Results for: ${result.collection.info.name}${colors.reset}`);

  if (result.run.executions) {
    result.run.executions.forEach((execution, index) => {
      const item = execution.item;
      const response = execution.response;
      const assertions = execution.assertions || [];

      const hasFailedAssertions = assertions.some(a => !a.passed);
      const statusColor = hasFailedAssertions ? colors.red : colors.green;

      console.log(`\n${colors.bright}${statusColor}${index + 1}. ${item.name}${colors.reset}`);
      console.log(`  Request: ${execution.request.method} ${execution.request.url.toString()}`);
      console.log(`  Status: ${response ? response.code : 'N/A'} ${response ? response.status : ''}`);
      console.log(`  Response Time: ${response ? response.responseTime : 'N/A'}ms`);

      console.log(`\n  ${colors.bright}Assertions:${colors.reset}`);
      assertions.forEach(assertion => {
        const assertionColor = assertion.passed ? colors.green : colors.red;
        console.log(`    ${assertionColor}${assertion.assertion}: ${assertion.passed ? 'Passed' : 'Failed'}${colors.reset}`);
        if (!assertion.passed && assertion.error) {
          console.log(`      ${colors.red}Error: ${assertion.error.message}${colors.reset}`);
        }
      });

      if (response && response.body) {
        console.log(`\n  ${colors.bright}Response Body:${colors.reset}`);
        try {
          const responseBody = JSON.parse(response.body);
          console.log('    ' + JSON.stringify(responseBody, null, 2).replace(/\n/g, '\n    '));
        } catch (e) {
          console.log(`    ${response.body}`);
        }
      }
    });
  }
}

// Function to analyze a specific test result file
function analyzeTestResult(filePath) {
  const result = readTestResults(filePath);
  if (!result) return;

  printTestSummary(result);

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  rl.question(`\n${colors.yellow}Do you want to see detailed results? (y/n) ${colors.reset}`, (answer) => {
    if (answer.toLowerCase() === 'y') {
      printDetailedTestResults(result);
    }
    rl.close();
  });
}

// Function to analyze all test result files
function analyzeAllTestResults() {
  const files = listTestResultFiles();
  if (files.length === 0) {
    console.log(`${colors.yellow}No test result files found in test-results directory${colors.reset}`);
    return;
  }

  console.log(`${colors.bright}${colors.cyan}Available Test Result Files:${colors.reset}`);
  files.forEach((file, index) => {
    console.log(`  ${index + 1}. ${path.basename(file)}`);
  });

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  rl.question(`\n${colors.yellow}Enter the number of the file to analyze (or 'all' for summary of all): ${colors.reset}`, (answer) => {
    if (answer.toLowerCase() === 'all') {
      files.forEach(file => {
        const result = readTestResults(file);
        if (result) {
          console.log(`\n${colors.bright}${colors.cyan}File: ${path.basename(file)}${colors.reset}`);
          printTestSummary(result);
        }
      });
      rl.close();
    } else {
      const fileIndex = parseInt(answer) - 1;
      if (fileIndex >= 0 && fileIndex < files.length) {
        rl.close();
        analyzeTestResult(files[fileIndex]);
      } else {
        console.log(`${colors.red}Invalid selection${colors.reset}`);
        rl.close();
      }
    }
  });
}

// Main function
function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  if (!command || command === 'help') {
    console.log(`\n${colors.bright}${colors.cyan}Test Result Analyzer${colors.reset}`);
    console.log(`\nUsage: node analyze_test_results.js [command] [options]`);
    console.log(`\nCommands:`);
    console.log(`  list                 List all test result files`);
    console.log(`  analyze <file>       Analyze a specific test result file`);
    console.log(`  analyze-all          Analyze all test result files`);
    console.log(`  help                 Show this help message`);
    return;
  }

  if (command === 'list') {
    const files = listTestResultFiles();
    console.log(`${colors.bright}${colors.cyan}Test Result Files:${colors.reset}`);
    files.forEach((file, index) => {
      console.log(`  ${index + 1}. ${path.basename(file)}`);
    });
    return;
  }

  if (command === 'analyze') {
    const fileName = args[1];
    if (!fileName) {
      console.log(`${colors.red}Error: No file specified${colors.reset}`);
      return;
    }

    const filePath = path.join(__dirname, 'test-results', fileName);
    if (!fs.existsSync(filePath)) {
      console.log(`${colors.red}Error: File not found: ${filePath}${colors.reset}`);
      return;
    }

    analyzeTestResult(filePath);
    return;
  }

  if (command === 'analyze-all') {
    analyzeAllTestResults();
    return;
  }

  console.log(`${colors.red}Error: Unknown command: ${command}${colors.reset}`);
}

main();
