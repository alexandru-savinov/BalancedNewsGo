package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run check_coverage.go <coverage_file> <threshold>")
		os.Exit(1)
	}

	coverageFile := os.Args[1]
	thresholdStr := os.Args[2]

	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		fmt.Printf("Invalid threshold %s: %v\n", thresholdStr, err)
		os.Exit(1)
	}

	file, err := os.Open(coverageFile) // #nosec G304 - coverageFile is from command line argument, controlled input
	if err != nil {
		fmt.Printf("Cannot open coverage file %s: %v\n", coverageFile, err)
		os.Exit(1)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
		}
	}()

	var totalCoverage float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				coverageStr := strings.TrimSuffix(fields[len(fields)-1], "%")
				totalCoverage, err = strconv.ParseFloat(coverageStr, 64)
				if err != nil {
					fmt.Printf("Cannot parse coverage percentage: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
	}

	fmt.Printf("Total coverage: %.2f%%\n", totalCoverage)

	if totalCoverage < threshold {
		fmt.Printf("Coverage %.2f%% is below threshold %.2f%%\n", totalCoverage, threshold)
		os.Exit(1)
	}

	fmt.Printf("Coverage %.2f%% meets or exceeds threshold %.2f%%\n", totalCoverage, threshold)
}
