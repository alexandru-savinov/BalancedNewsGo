#!/bin/bash

# Directory to store test outputs (allow override via env var)
RESULTS_DIR="${RESULTS_DIR:-test-results}"
# Create test-results directory if it doesn't exist
mkdir -p "$RESULTS_DIR"

# Function to start server in background and capture PID
start_server() {
  local log_file=$1
  echo "Starting the server (logging to $log_file)..."
  go run ./cmd/server > "$log_file" 2>&1 &
  SERVER_PID=$!
  echo "Server PID: $SERVER_PID"
  echo "Waiting for server health check..."
  local retries=0
  until curl --silent http://localhost:8080/health > /dev/null; do
    if ! ps -p $SERVER_PID > /dev/null; then
      echo "Server process $SERVER_PID terminated unexpectedly"
      exit 1
    fi
    retries=$((retries+1))
    if [ $retries -ge 10 ]; then
      echo "Server did not become healthy in time"
      stop_server
      exit 1
    fi
    sleep 1
  done
  echo "Server started and healthy"
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
  # include Postman globals for baseUrl
  newman_cmd="$newman_cmd -g postman/NewsBalancer.postman_globals.json"
  # prevent hanging by setting request and script timeouts
  newman_cmd="$newman_cmd --timeout-request 60000 --timeout 120000"
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

    # Remove existing database for a fresh start
    rm -f news.db news.db-wal news.db-shm
    echo "Deleted old database files"

    # Reset test database to a clean state
    echo "Resetting test database..." >> "$all_log_file"
    go run cmd/reset_test_db/main.go >> "$all_log_file" 2>&1
    if [ $? -ne 0 ]; then echo "DB reset FAILED. Check $all_log_file"; exit 1; fi

    start_server "$server_log"

    echo "====== NewsBalancer Full API Test Run - $(date) ======" > "$all_log_file"
    echo "Server log: $server_log" >> "$all_log_file"
    echo "" >> "$all_log_file"

    echo "===== Running Unified Backend Tests (with retries) =====" >> "$all_log_file"
    echo "Running Newman: postman/unified_backend_tests.json with environment postman/newman_environment.json" | tee -a "$all_log_file"
    # Retry logic for transient failures
    attempt=1
    max_attempts=3
    while [ $attempt -le $max_attempts ]; do
      echo "Attempt $attempt of $max_attempts" >> "$all_log_file"
      npx newman run postman/unified_backend_tests.json \
        -e postman/newman_environment.json \
        -g postman/NewsBalancer.postman_globals.json \
        --timeout-request 60000 --timeout 120000 \
        --reporters cli,json \
        --reporter-json-export="$RESULTS_DIR/unified_results_${timestamp}.json" 2>&1 | tee -a "$all_log_file" && break
      echo "Unified tests failed on attempt $attempt" >> "$all_log_file"
      attempt=$((attempt+1))
      if [ $attempt -le $max_attempts ]; then
        echo "Retrying unified tests..." >> "$all_log_file"
        sleep 2
      else
        echo "Unified tests FAILED after $max_attempts attempts. Check $all_log_file" >> "$all_log_file"
        exit 1
      fi
    done
    echo "Unified tests succeeded on attempt $attempt" >> "$all_log_file"

    echo "" >> "$all_log_file"
    echo "===== Generating Test Report =====" >> "$all_log_file"
    generate_report >> "$all_log_file" 2>&1
    if [ $? -ne 0 ]; then echo "Report generation FAILED. Check $all_log_file"; exit 1; fi

    echo "" >> "$all_log_file"
    echo "===== All tests completed SUCCESSFULLY =====" >> "$all_log_file"
    echo "Test log saved to: $all_log_file"
    cat "$all_log_file"

    stop_server
    exit 0
}

# --- Main Execution ---
COMMAND=$1
if [ -z "$COMMAND" ]; then
    COMMAND="help"
fi

case $COMMAND in
    "backend")
        echo "Running backend fixes tests..."
        run_newman_test "backend_fixes" "postman/backend_fixes_tests_updated.json" "postman/local_environment.json"
        # Add SSE check if needed: node scripts/test_sse_progress.js
        ;;
    "api")
        echo "Running basic API tests..."
        run_newman_test "api_tests" "postman/newsbalancer_api_tests.json" "postman/local_environment.json"
        ;;    "unified")
        echo "Running unified tests..."
        run_newman_test "unified_tests" "postman/unified_backend_tests.json" ""
        ;;
    "essential")
        echo "Running unified tests (essential is deprecated, use 'unified')..."
        run_newman_test "unified_tests" "postman/unified_backend_tests.json" ""
        ;;
    "debug")
        echo "Running debug tests..."
        run_newman_test "debug_tests" "postman/debug_collection.json" "postman/debug_environment.json"
        ;;
    "all")
        run_all_tests_suite # Use the dedicated function for the multi-stage 'all' run
        ;;
    "confidence")
        if [ -f postman/backup/confidence_validation_tests.json ]; then
            echo "Running confidence validation tests..."
            run_newman_test "confidence_tests" "postman/backup/confidence_validation_tests.json" ""
        else
             echo "Confidence test collection not found: postman/backup/confidence_validation_tests.json"
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
        echo "  backend        - Run backend fixes/integration tests (Newman: backend_fixes_tests_updated.json)"
        echo "  api            - Run basic API tests (Newman: newsbalancer_api_tests.json)"
        echo "  unified        - Run unified tests (Newman: unified_backend_tests.json)"
        echo "  essential      - [DEPRECATED] Alias for 'unified' command"
        echo "  debug          - Run debug tests (Newman: debug_collection.json)"
        echo "  all            - Run unified, extended, and confidence tests (Multiple Newman collections)"
        echo "  confidence     - Run confidence validation tests (Newman: confidence_validation_tests.json) [If available]"
        echo
        echo "  report         - Generate HTML test report from results"
        echo "  analyze        - Analyze test results via CLI"
        echo "  list           - List existing test result files"
        echo "  clean          - Clean (delete) test result files (*.json, *.html, *.log)"
        echo "  help           - Show this help message"
        echo
        echo "Notes:"
        echo "  - Test commands (backend, api, unified, debug, confidence) typically start/stop the Go server."
        echo "  - The 'all' command runs multiple test suites sequentially without restarting the server between them."
        echo "  - Requires Node.js and Newman ('npm install -g newman')."
        echo
        ;;
esac
exit 0
