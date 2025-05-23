#!/bin/bash

# Directory to store test outputs (allow override via env var)
RESULTS_DIR="${RESULTS_DIR:-test-results}"
# Provide dummy LLM_API_KEY if not set to allow server startup during tests
export LLM_API_KEY="${LLM_API_KEY:-test-key}"
# Create test-results directory if it doesn't exist
mkdir -p "$RESULTS_DIR"

# Function to start server in background and capture PID
start_server() {
  local log_file=$1
  echo "Starting the server (logging to $log_file)..."
  go run cmd/server/main.go > "$log_file" 2>&1 &
  SERVER_PID=$!
  echo "Server PID: $SERVER_PID"
  echo "Waiting for the server to start..."
  sleep 5
}

# Function to stop the server
stop_server() {
  if [ ! -z "$SERVER_PID" ]; then
    echo "Stopping the server (PID: $SERVER_PID)..."
    # Try killing gently first, then force if needed
    kill $SERVER_PID 2>/dev/null || kill -9 $SERVER_PID 2>/dev/null
    # Wait a moment for cleanup
    sleep 1
    # Verify killed
    if ps -p $SERVER_PID > /dev/null; then
       echo "Warning: Server PID $SERVER_PID might still be running."
    fi
    SERVER_PID=""
  else
    echo "No server PID recorded to stop."
    # Fallback: kill any running instances (use with caution)
    # pkill -f "go run cmd/server/main.go"
    # pkill -f newbalancer_server # If running compiled binary
  fi
}

# Trap EXIT signal to ensure server is stopped
trap stop_server EXIT

# Function to run a Newman test suite
# $1: Test name (for logging/filenames)
# $2: Newman collection file path
# $3: Newman environment file path (optional, use "" if none)
run_newman_test() {
  local test_name=$1
  local collection_file=$2
  local env_file=$3
  local server_log="$RESULTS_DIR/server_${test_name}.log"
  local result_file="$RESULTS_DIR/${test_name}_results.json"

  start_server "$server_log"

  echo "Running Newman test: $collection_file"
  local newman_cmd="npx newman run \"$collection_file\""
  if [ ! -z "$env_file" ]; then
    newman_cmd="$newman_cmd -e \"$env_file\""
  fi
  newman_cmd="$newman_cmd --reporters cli,json --reporter-json-export \"$result_file\""

  echo "Executing: $newman_cmd"
  eval $newman_cmd
  local newman_exit_code=$?

  # Server is stopped via EXIT trap

  if [ $newman_exit_code -ne 0 ]; then
    echo "$test_name tests FAILED (Exit Code: $newman_exit_code). Check output above and $result_file."
    exit $newman_exit_code
  fi

  echo "$test_name tests completed successfully. Results: $result_file"
}

# Function to generate report
generate_report() {
    echo "Generating HTML test report..."
    node scripts/generate_test_report.js
    if [ $? -ne 0 ]; then
        echo "Failed to generate report."
        exit 1
    fi
    echo "Report generated at $RESULTS_DIR/test_report.html"
}

# Function to clean results
clean_results() {
    echo "Cleaning test results..."
    rm -f "$RESULTS_DIR"/*.json
    rm -f "$RESULTS_DIR"/*.html
    rm -f "$RESULTS_DIR"/*.log
    echo "Test results cleaned."
}

# Function to run all tests (matches run_all_tests.cmd logic)
run_all_tests_suite() {
    echo "Running all test suites..."
    local timestamp=$(date +%Y%m%d%H%M%S)
    local all_log_file="$RESULTS_DIR/all_tests_run_${timestamp}.log"
    local server_log="$RESULTS_DIR/server_all_tests.log"

    start_server "$server_log"

    echo "====== NewsBalancer Full API Test Run - $(date) ======" > "$all_log_file"
    echo "Server log: $server_log" >> "$all_log_file"
    echo "" >> "$all_log_file"

    echo "===== Running Essential Tests =====" >> "$all_log_file"
    echo "Running Newman: postman/backup/essential_rescoring_tests.json" | tee -a "$all_log_file"
    npx newman run postman/backup/essential_rescoring_tests.json \
      --reporters cli,json \
      --reporter-json-export="$RESULTS_DIR/essential_results_${timestamp}.json" >> "$all_log_file" 2>&1
    if [ $? -ne 0 ]; then echo "Essential tests FAILED. Check $all_log_file"; exit 1; fi

    echo "" >> "$all_log_file"
    echo "===== Running Extended Tests =====" >> "$all_log_file"
    echo "Running Newman: postman/backup/extended_rescoring_collection.json" | tee -a "$all_log_file"
    npx newman run postman/backup/extended_rescoring_collection.json \
      --reporters cli,json \
      --reporter-json-export="$RESULTS_DIR/extended_results_${timestamp}.json" >> "$all_log_file" 2>&1
     if [ $? -ne 0 ]; then echo "Extended tests FAILED. Check $all_log_file"; exit 1; fi

    if [ -f postman/backup/confidence_validation_tests.json ]; then
      echo "" >> "$all_log_file"
      echo "===== Running Confidence Validation Tests =====" >> "$all_log_file"
      echo "Running Newman: postman/backup/confidence_validation_tests.json" | tee -a "$all_log_file"
      npx newman run postman/backup/confidence_validation_tests.json \
        --reporters cli,json \
        --reporter-json-export="$RESULTS_DIR/confidence_results_${timestamp}.json" >> "$all_log_file" 2>&1
      if [ $? -ne 0 ]; then echo "Confidence tests FAILED. Check $all_log_file"; exit 1; fi
    fi

    echo "" >> "$all_log_file"
    echo "===== Generating Test Report =====" >> "$all_log_file"
    generate_report >> "$all_log_file" 2>&1
    if [ $? -ne 0 ]; then echo "Report generation FAILED. Check $all_log_file"; exit 1; fi

    echo "" >> "$all_log_file"
    echo "===== All tests completed SUCCESSFULLY =====" >> "$all_log_file"
    echo "Test log saved to: $all_log_file"
    cat "$all_log_file"

    stop_server
}

# --- Main Execution --- 
COMMAND=$1
if [ -z "$COMMAND" ]; then
    COMMAND="help"
fi

case $COMMAND in
    "backend")
        echo "Running backend fixes tests..."
        run_newman_test "backend_fixes" "postman/unified_backend_tests.json" "postman/local_environment.json"
        # Add SSE check if needed: node scripts/test_sse_progress.js 
        ;;
    "api")
        echo "Running basic API tests..."
        run_newman_test "api_tests" "postman/unified_backend_tests.json" "postman/local_environment.json"
        ;;
    "essential")
        echo "Running essential rescoring tests..."
        run_newman_test "essential_tests" "postman/unified_backend_tests.json" ""
        ;;
    "debug")
        echo "Running debug tests..."
        run_newman_test "debug_tests" "postman/unified_backend_tests.json" "postman/debug_environment.json"
        ;;
    "all")
        run_all_tests_suite # Use the dedicated function for the multi-stage 'all' run
        ;;
    "confidence")
        if [ -f postman/confidence_validation_tests.json ]; then
            echo "Running confidence validation tests..."
            run_newman_test "confidence_tests" "postman/confidence_validation_tests.json" ""
        else
             echo "Confidence test collection not found: postman/confidence_validation_tests.json"
        fi
        ;;
    "report")
        generate_report
        ;;
    "clean")
        clean_results
        ;;
    "analyze")
        echo "Analyzing test results..."
        node scripts/analyze_test_results.js analyze-all
        ;;
    "list")
        echo "Listing test result files..."
        node scripts/analyze_test_results.js list
        ;;
    "help"|*)
        echo
        echo "NewBalancer Go Testing CLI"
        echo "========================"
        echo
        echo "Usage: ./test.sh [command]"
        echo
        echo "Available commands:"
        echo
        echo "  backend        - Run backend fixes/integration tests (Newman: unified_backend_tests.json)"
        echo "  api            - Run basic API tests (Newman: unified_backend_tests.json)"
        echo "  essential      - Run essential rescoring tests (Newman: unified_backend_tests.json)"
        echo "  debug          - Run debug tests (Newman: unified_backend_tests.json)"
        echo "  all            - Run essential, extended, and confidence tests (Multiple Newman collections)"
        echo "  confidence     - Run confidence validation tests (Newman: confidence_validation_tests.json) [If available]"
        echo
        echo "  report         - Generate HTML test report from results"
        echo "  analyze        - Analyze test results via CLI"
        echo "  list           - List existing test result files"
        echo "  clean          - Clean (delete) test result files (*.json, *.html, *.log)"
        echo "  help           - Show this help message"
        echo
        echo "Notes:"
        echo "  - Test commands (backend, api, essential, debug, confidence) typically start/stop the Go server."
        echo "  - The 'all' command runs multiple test suites sequentially without restarting the server between them."
        echo "  - Requires Node.js and Newman ('npm install -g newman')."
        echo
        ;;
esac
