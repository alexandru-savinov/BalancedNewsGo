#!/bin/bash

# Code Coverage Analysis Script for NewsBalancer
# This script runs tests with coverage analysis and generates reports

set -e

# Configuration
COVERAGE_DIR="./coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"
COVERAGE_JSON="$COVERAGE_DIR/coverage.json"
MIN_COVERAGE=80

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}NewsBalancer Code Coverage Analysis${NC}"
echo -e "${BLUE}===================================${NC}"
echo ""

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Clean previous coverage data
rm -f "$COVERAGE_FILE" "$COVERAGE_HTML" "$COVERAGE_JSON"

echo -e "${YELLOW}Running tests with coverage...${NC}"

# Set environment variables for testing
export NO_AUTO_ANALYZE=true
export DATABASE_URL="${DATABASE_URL:-postgres://localhost:5432/newsbalancer_test?sslmode=disable}"

# Run tests with coverage
go test -v -race -coverprofile="$COVERAGE_FILE" -covermode=atomic ./... 2>&1 | tee "$COVERAGE_DIR/test-output.log"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo -e "${RED}Error: Coverage file not generated${NC}"
    exit 1
fi

echo -e "${GREEN}Tests completed successfully${NC}"
echo ""

# Generate HTML coverage report
echo -e "${YELLOW}Generating HTML coverage report...${NC}"
go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML"
echo -e "${GREEN}HTML report generated: $COVERAGE_HTML${NC}"

# Generate detailed coverage analysis
echo -e "${YELLOW}Analyzing coverage by package...${NC}"

# Create a detailed coverage report
cat > "$COVERAGE_DIR/coverage-report.txt" << EOF
# NewsBalancer Code Coverage Report
Generated: $(date)

## Overall Coverage
EOF

# Calculate overall coverage
TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "total:" | awk '{print $3}' | sed 's/%//')

echo "Total Coverage: ${TOTAL_COVERAGE}%" >> "$COVERAGE_DIR/coverage-report.txt"
echo "" >> "$COVERAGE_DIR/coverage-report.txt"

# Package-level coverage
echo "## Package Coverage" >> "$COVERAGE_DIR/coverage-report.txt"
echo "" >> "$COVERAGE_DIR/coverage-report.txt"

go tool cover -func="$COVERAGE_FILE" | grep -v "total:" | while read line; do
    if [[ $line == *".go:"* ]]; then
        package=$(echo "$line" | awk -F'/' '{print $(NF-1)"/"$NF}' | awk -F':' '{print $1}')
        coverage=$(echo "$line" | awk '{print $3}')
        echo "- $package: $coverage" >> "$COVERAGE_DIR/coverage-report.txt"
    fi
done

# Function coverage (functions with low coverage)
echo "" >> "$COVERAGE_DIR/coverage-report.txt"
echo "## Functions with Low Coverage (<50%)" >> "$COVERAGE_DIR/coverage-report.txt"
echo "" >> "$COVERAGE_DIR/coverage-report.txt"

go tool cover -func="$COVERAGE_FILE" | awk '$3 != "100.0%" && $3 != "0.0%" {
    coverage = $3
    gsub(/%/, "", coverage)
    if (coverage < 50 && coverage > 0) {
        print "- " $1 " " $2 ": " $3
    }
}' >> "$COVERAGE_DIR/coverage-report.txt"

# Generate JSON report for programmatic access
echo -e "${YELLOW}Generating JSON coverage report...${NC}"
cat > "$COVERAGE_JSON" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "total_coverage": $TOTAL_COVERAGE,
  "threshold": $MIN_COVERAGE,
  "passed": $(if (( $(echo "$TOTAL_COVERAGE >= $MIN_COVERAGE" | bc -l) )); then echo "true"; else echo "false"; fi),
  "packages": [
EOF

# Add package coverage to JSON
FIRST=true
go tool cover -func="$COVERAGE_FILE" | grep -v "total:" | while read line; do
    if [[ $line == *".go:"* ]]; then
        package=$(echo "$line" | awk -F'/' '{print $(NF-1)"/"$NF}' | awk -F':' '{print $1}')
        coverage=$(echo "$line" | awk '{print $3}' | sed 's/%//')
        
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo "," >> "$COVERAGE_JSON"
        fi
        
        echo "    {" >> "$COVERAGE_JSON"
        echo "      \"package\": \"$package\"," >> "$COVERAGE_JSON"
        echo "      \"coverage\": $coverage" >> "$COVERAGE_JSON"
        echo -n "    }" >> "$COVERAGE_JSON"
    fi
done

echo "" >> "$COVERAGE_JSON"
echo "  ]" >> "$COVERAGE_JSON"
echo "}" >> "$COVERAGE_JSON"

echo -e "${GREEN}JSON report generated: $COVERAGE_JSON${NC}"
echo ""

# Display summary
echo -e "${BLUE}Coverage Summary:${NC}"
echo -e "  Total Coverage: ${TOTAL_COVERAGE}%"
echo -e "  Minimum Required: ${MIN_COVERAGE}%"

if (( $(echo "$TOTAL_COVERAGE >= $MIN_COVERAGE" | bc -l) )); then
    echo -e "  Status: ${GREEN}PASSED${NC}"
    COVERAGE_STATUS=0
else
    echo -e "  Status: ${RED}FAILED${NC}"
    echo -e "  ${RED}Coverage is below the minimum threshold of ${MIN_COVERAGE}%${NC}"
    COVERAGE_STATUS=1
fi

echo ""
echo -e "${BLUE}Generated Files:${NC}"
echo -e "  Coverage Data: $COVERAGE_FILE"
echo -e "  HTML Report: $COVERAGE_HTML"
echo -e "  JSON Report: $COVERAGE_JSON"
echo -e "  Text Report: $COVERAGE_DIR/coverage-report.txt"
echo -e "  Test Output: $COVERAGE_DIR/test-output.log"

echo ""
echo -e "${YELLOW}To view the HTML report, open: file://$(pwd)/$COVERAGE_HTML${NC}"

# Check if we should fail based on coverage
if [ "$COVERAGE_STATUS" -ne 0 ]; then
    echo ""
    echo -e "${RED}Coverage check failed. Please improve test coverage before proceeding.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Coverage analysis completed successfully!${NC}"
