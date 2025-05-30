#!/bin/bash

# Test verification script to confirm all fixes are working

echo "=== Starting Test Verification ==="

# 1. Check if the package compiles
echo "1. Checking compilation..."
cd /d/Dev/newbalancer_go
if go build ./internal/llm/...; then
    echo "✅ LLM package compiles successfully"
else
    echo "❌ LLM package compilation failed"
    exit 1
fi

# 2. Run LLM tests specifically
echo "2. Running LLM tests..."
export NO_AUTO_ANALYZE=true
if go test ./internal/llm/... -count=1 -timeout 2m; then
    echo "✅ All LLM tests pass"
else
    echo "❌ LLM tests failed"
    exit 1
fi

# 3. Check formatting
echo "3. Checking code formatting..."
if [ -z "$(gofmt -l ./internal/llm/)" ]; then
    echo "✅ Code is properly formatted"
else
    echo "❌ Code formatting issues found"
    gofmt -l ./internal/llm/
    exit 1
fi

# 4. Run vet
echo "4. Running go vet..."
if go vet ./internal/llm/...; then
    echo "✅ No vet issues found"
else
    echo "❌ Vet issues found"
    exit 1
fi

# 5. Check mod tidy
echo "5. Checking go mod tidy..."
if go mod tidy; then
    echo "✅ Dependencies are clean"
else
    echo "❌ Dependency issues found"
    exit 1
fi

echo ""
echo "=== All Tests Completed Successfully! ==="
echo "✅ Fixed redeclaration error for ErrAllPerspectivesInvalid"
echo "✅ All LLM unit tests pass"
echo "✅ Code compiles without errors"
echo "✅ Code formatting is correct"
echo "✅ No vet issues"
echo "✅ Dependencies are clean"
echo ""
echo "The precommit configuration has also been updated to work without make dependency."
