#!/bin/bash

# Create test-results directory if it doesn't exist
mkdir -p test-results

# Start the server in the background
go run cmd/server/main.go &
SERVER_PID=$!

# Wait for the server to start
echo "Waiting for server to start..."
sleep 5

# Run the tests
echo "Running tests..."
npx newman run memory-bank/essential_rescoring_tests.json --reporters cli,json --reporter-json-export test-results/essential_rescoring_tests.json

# Capture the exit code
TEST_EXIT_CODE=$?

# Kill the server
echo "Stopping server..."
kill $SERVER_PID

# Display the results
echo "Test execution completed with exit code $TEST_EXIT_CODE. Results saved to test-results/essential_rescoring_tests.json"

# Exit with the test exit code
exit $TEST_EXIT_CODE