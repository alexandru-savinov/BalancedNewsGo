#!/bin/bash
cd /d/Dev/newbalancer_go
export NO_AUTO_ANALYZE=true
echo "Testing internal/llm package..."
go test ./internal/llm -count=1 -short -timeout=30s -v
echo "Exit code: $?"
