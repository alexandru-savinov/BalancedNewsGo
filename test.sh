#!/bin/bash

# Create test-results directory if it doesn't exist
mkdir -p test-results

# Function to run backend tests
run_backend_tests() {
    echo "Starting the server..."
    go run cmd/server/main.go > test-results/server.log 2>&1 &
    SERVER_PID=$!
    
    echo "Waiting for the server to start..."
    sleep 5
    
    echo "Running backend fixes tests..."
    npx newman run postman/backup/backend_fixes_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/backend_fixes_tests.json
    
    echo "Stopping the server..."
    kill $SERVER_PID
    
    echo "Backend tests completed."
}

# Function to run all tests
run_all_tests() {
    echo "Running backend fixes tests..."
    run_backend_tests
    
    echo "Running rescoring tests..."
    go run cmd/server/main.go > test-results/server_rescoring.log 2>&1 &
    SERVER_PID=$!
    sleep 5
    npx newman run postman/backup/postman_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/postman_rescoring_collection.json
    kill $SERVER_PID
    
    echo "Running essential rescoring tests..."
    go run cmd/server/main.go > test-results/server_essential.log 2>&1 &
    SERVER_PID=$!
    sleep 5
    npx newman run postman/backup/essential_rescoring_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/essential_rescoring_tests.json
    kill $SERVER_PID
    
    echo "Running extended rescoring tests..."
    go run cmd/server/main.go > test-results/server_extended.log 2>&1 &
    SERVER_PID=$!
    sleep 5
    npx newman run postman/backup/extended_rescoring_collection.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/extended_rescoring_collection.json
    kill $SERVER_PID
    
    echo "All tests completed."
}

# Function to run debug tests
run_debug_tests() {
    echo "Starting the server..."
    go run cmd/server/main.go > test-results/server_debug.log 2>&1 &
    SERVER_PID=$!
    
    echo "Waiting for the server to start..."
    sleep 5
    
    echo "Running debug tests..."
    npx newman run postman/debug_collection.json -e postman/debug_environment.json --reporters cli,json --reporter-json-export test-results/debug_tests.json
    
    echo "Stopping the server..."
    kill $SERVER_PID
    
    echo "Debug tests completed."
}

# Function to run confidence tests
run_confidence_tests() {
    echo "Running confidence validation tests..."
    go run cmd/server/main.go > test-results/server_confidence.log 2>&1 &
    SERVER_PID=$!
    sleep 5
    
    npx newman run postman/backup/confidence_validation_tests.json -e postman/local_environment.json --reporters cli,json --reporter-json-export test-results/confidence_tests.json
    
    echo "Stopping the server..."
    kill $SERVER_PID
    
    echo "Confidence tests completed."
}

# Function to generate report
generate_report() {
    echo "Generating HTML test report..."
    node generate_test_report.js
    echo "Report generated at test-results/test_report.html"
}

# Function to clean results
clean_results() {
    echo "Cleaning test results..."
    rm -f test-results/*.json
    rm -f test-results/*.html
    rm -f test-results/*.log
    echo "Test results cleaned."
}

# Parse command line arguments
COMMAND=$1
if [ -z "$COMMAND" ]; then
    COMMAND="help"
fi

case $COMMAND in
    "backend")
        run_backend_tests
        ;;
    "all")
        run_all_tests
        ;;
    "debug")
        run_debug_tests
        ;;
    "confidence")
        run_confidence_tests
        ;;
    "report")
        generate_report
        ;;
    "clean")
        clean_results
        ;;
    "analyze")
        echo "Analyzing test results..."
        node analyze_test_results.js analyze-all
        ;;
    "list")
        echo "Listing test result files..."
        node analyze_test_results.js list
        ;;
    "help"|*)
        echo
        echo "NewBalancer Go Testing CLI"
        echo "========================"
        echo
        echo "Available commands:"
        echo
        echo "  ./test.sh backend        - Run backend fixes tests"
        echo "  ./test.sh all            - Run all tests"
        echo "  ./test.sh debug          - Run debug tests"
        echo "  ./test.sh confidence     - Run confidence validation tests"
        echo "  ./test.sh report         - Generate HTML test report"
        echo "  ./test.sh analyze        - Analyze test results"
        echo "  ./test.sh list           - List test result files"
        echo "  ./test.sh clean          - Clean test results"
        echo "  ./test.sh help           - Show this help message"
        echo
        ;;
esac