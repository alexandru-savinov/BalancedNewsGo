#!/bin/bash

# NewsBalancer Performance Benchmark Script
# This script runs various performance benchmarks and generates reports

set -e

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
DB_URL="${DB_URL:-}"
OUTPUT_DIR="${OUTPUT_DIR:-./benchmark-results}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}NewsBalancer Performance Benchmark Suite${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Check if the API is running
echo -e "${YELLOW}Checking API availability...${NC}"
if ! curl -s "$BASE_URL/api/articles" > /dev/null; then
    echo -e "${RED}Error: API is not accessible at $BASE_URL${NC}"
    echo "Please ensure the NewsBalancer server is running"
    exit 1
fi
echo -e "${GREEN}API is accessible${NC}"
echo ""

# Build benchmark tool if it doesn't exist
if [ ! -f "./cmd/benchmark/benchmark" ]; then
    echo -e "${YELLOW}Building benchmark tool...${NC}"
    cd cmd/benchmark
    go build -o benchmark .
    cd ../..
    echo -e "${GREEN}Benchmark tool built${NC}"
fi

# Function to run a benchmark test
run_benchmark() {
    local test_name="$1"
    local users="$2"
    local requests="$3"
    local duration="$4"
    local output_file="$OUTPUT_DIR/${test_name}_${TIMESTAMP}"
    
    echo -e "${YELLOW}Running $test_name benchmark...${NC}"
    echo "  Users: $users, Requests per user: $requests, Duration: $duration"
    
    # Run benchmark and save results
    ./cmd/benchmark/benchmark \
        -test "$test_name" \
        -url "$BASE_URL" \
        -users "$users" \
        -requests "$requests" \
        -duration "$duration" \
        -db "$DB_URL" \
        -output json > "${output_file}.json"
    
    ./cmd/benchmark/benchmark \
        -test "$test_name" \
        -url "$BASE_URL" \
        -users "$users" \
        -requests "$requests" \
        -duration "$duration" \
        -db "$DB_URL" \
        -output csv > "${output_file}.csv"
    
    echo -e "${GREEN}$test_name benchmark completed${NC}"
    echo "  Results saved to: ${output_file}.json and ${output_file}.csv"
    echo ""
}

# Run different benchmark scenarios
echo -e "${BLUE}Starting benchmark tests...${NC}"
echo ""

# Light load test
run_benchmark "light-load" 5 20 "2m"

# Medium load test
run_benchmark "medium-load" 15 50 "3m"

# Heavy load test
run_benchmark "heavy-load" 30 100 "5m"

# Stress test
run_benchmark "stress-test" 50 200 "10m"

# Spike test (quick burst)
run_benchmark "spike-test" 100 10 "1m"

# Generate summary report
echo -e "${BLUE}Generating summary report...${NC}"
SUMMARY_FILE="$OUTPUT_DIR/benchmark_summary_${TIMESTAMP}.md"

cat > "$SUMMARY_FILE" << EOF
# NewsBalancer Performance Benchmark Report

**Generated:** $(date)
**API URL:** $BASE_URL
**Test Suite:** Automated Performance Benchmarks

## Test Results Summary

EOF

# Process JSON results to create summary
for json_file in "$OUTPUT_DIR"/*_${TIMESTAMP}.json; do
    if [ -f "$json_file" ]; then
        test_name=$(basename "$json_file" .json | sed "s/_${TIMESTAMP}//")
        
        # Extract key metrics using jq if available, otherwise use basic parsing
        if command -v jq > /dev/null; then
            total_requests=$(jq -r '.total_requests' "$json_file")
            success_rate=$(jq -r '.successful_requests' "$json_file")
            error_rate=$(jq -r '.error_rate' "$json_file")
            avg_latency=$(jq -r '.average_latency' "$json_file")
            p95_latency=$(jq -r '.p95_latency' "$json_file")
            rps=$(jq -r '.requests_per_second' "$json_file")
            
            cat >> "$SUMMARY_FILE" << EOF
### $test_name

- **Total Requests:** $total_requests
- **Successful Requests:** $success_rate
- **Error Rate:** ${error_rate}%
- **Average Latency:** $avg_latency
- **95th Percentile Latency:** $p95_latency
- **Requests per Second:** $rps

EOF
        else
            echo "### $test_name" >> "$SUMMARY_FILE"
            echo "" >> "$SUMMARY_FILE"
            echo "*(jq not available - see JSON file for detailed results)*" >> "$SUMMARY_FILE"
            echo "" >> "$SUMMARY_FILE"
        fi
    fi
done

cat >> "$SUMMARY_FILE" << EOF

## Files Generated

EOF

# List all generated files
for file in "$OUTPUT_DIR"/*_${TIMESTAMP}.*; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        echo "- \`$filename\`" >> "$SUMMARY_FILE"
    fi
done

cat >> "$SUMMARY_FILE" << EOF

## Recommendations

Based on the benchmark results:

1. **Performance Baseline:** Use these results as a baseline for future performance comparisons
2. **Monitoring:** Set up alerts based on the 95th percentile latency values
3. **Scaling:** Consider horizontal scaling if error rates exceed 1% under normal load
4. **Optimization:** Focus on endpoints with highest latency for optimization efforts

## Next Steps

1. Review individual test results in the JSON/CSV files
2. Compare results with previous benchmark runs
3. Investigate any performance regressions
4. Update monitoring thresholds based on these results

EOF

echo -e "${GREEN}Summary report generated: $SUMMARY_FILE${NC}"
echo ""

# Display quick summary
echo -e "${BLUE}Quick Summary:${NC}"
echo "  Output directory: $OUTPUT_DIR"
echo "  Files generated: $(ls -1 "$OUTPUT_DIR"/*_${TIMESTAMP}.* | wc -l)"
echo "  Summary report: $SUMMARY_FILE"
echo ""

echo -e "${GREEN}All benchmarks completed successfully!${NC}"
echo -e "${YELLOW}Review the generated files for detailed performance analysis.${NC}"
