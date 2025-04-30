#!/bin/bash

# Navigate to the project root
cd "$(dirname "$0")"

# Set the environment to test mode
export GIN_MODE=test

# Run the tests for critical API handlers
go test -v -coverprofile=api_coverage_critical.out ./internal/api -run "^Test(CreateArticleHandler|GetArticlesHandler|GetArticleByID|BiasHandler|SummaryHandler)"

# Generate a coverage report if tests passed
if [ $? -eq 0 ]; then
  echo "Tests passed! Generating coverage report..."
  go tool cover -html=api_coverage_critical.out -o api_coverage_critical.html
  echo "Coverage report saved to api_coverage_critical.html"
else
  echo "Tests failed!"
fi