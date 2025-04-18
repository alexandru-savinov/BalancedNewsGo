#!/bin/bash

# Clean and rebuild the application
echo "Cleaning and rebuilding..."
go clean
go build -o newbalancer_server.exe cmd/server/main.go

# Start the server
echo "Starting the server..."
./newbalancer_server.exe