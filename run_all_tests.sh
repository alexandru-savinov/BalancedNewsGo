#!/bin/bash

# Set up output directory
RESULTS_DIR="./test-results"
mkdir -p $RESULTS_DIR
TIMESTAMP=$(date +"%Y%m%d%H%M%S")
LOG_FILE="${RESULTS_DIR}/test_run_${TIMESTAMP}.log"

echo "====== NewsBalancer API Test Run - $(date) ======" | tee -a $LOG_FILE
echo "Running with NODE_ENV=${NODE_ENV:-development}" | tee -a $LOG_FILE

# Check for .env file
if [ -f ".env" ]; then
  echo "Using .env configuration file" | tee -a $LOG_FILE
else
  echo "Warning: No .env file found - using default environment variables" | tee -a $LOG_FILE
fi

# Run essential tests first
echo -e "\n===== Running Essential Tests =====" | tee -a $LOG_FILE
npx newman run postman/backup/essential_rescoring_tests.json \
  --reporters cli,json \
  --reporter-json-export="${RESULTS_DIR}/essential_results.json" | tee -a $LOG_FILE

# Run extended tests 
echo -e "\n===== Running Extended Tests =====" | tee -a $LOG_FILE
npx newman run postman/backup/extended_rescoring_collection.json \
  --reporters cli,json \
  --reporter-json-export="${RESULTS_DIR}/extended_results.json" | tee -a $LOG_FILE

# Run confidence validation tests if they exist
if [ -f "postman/backup/confidence_validation_tests.json" ]; then
  echo -e "\n===== Running Confidence Validation Tests =====" | tee -a $LOG_FILE
  npx newman run postman/backup/confidence_validation_tests.json \
    --reporters cli,json \
    --reporter-json-export="${RESULTS_DIR}/confidence_results.json" | tee -a $LOG_FILE
fi

# Generate test summary report
echo -e "\n===== Generating Test Report =====" | tee -a $LOG_FILE
node analyze_test_results.js | tee -a $LOG_FILE

echo -e "\n===== All tests completed =====" | tee -a $LOG_FILE
echo "Test log saved to: $LOG_FILE"