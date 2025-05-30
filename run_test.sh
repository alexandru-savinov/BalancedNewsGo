#!/bin/bash
cd "$(dirname "$0")"
export NO_AUTO_ANALYZE=true

echo "Running LLM tests..."
go test -v ./internal/llm -timeout 30s > test_output.txt 2>&1
cat test_output.txt
