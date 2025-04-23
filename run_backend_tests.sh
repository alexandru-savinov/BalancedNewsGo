#!/bin/bash

# Start the server in the background
echo "Starting the server..."
go run cmd/server/main.go &
SERVER_PID=$!

# Wait for the server to start
echo "Waiting for the server to start..."
sleep 5

# Run the Newman tests
echo "Running Newman tests..."
npx newman run postman/backend_fixes_tests.json -e postman/local_environment.json

# Kill the server
echo "Stopping the server..."
kill $SERVER_PID

echo "Tests completed."