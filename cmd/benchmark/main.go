package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/benchmark"
)

func main() {
	var (
		configFile = flag.String("config", "benchmark-config.json", "Path to benchmark configuration file")
		testName   = flag.String("test", "load-test", "Name of the test to run")
		baseURL    = flag.String("url", "http://localhost:8080", "Base URL of the API to test")
		users      = flag.Int("users", 10, "Number of concurrent users")
		requests   = flag.Int("requests", 100, "Number of requests per user")
		duration   = flag.Duration("duration", 5*time.Minute, "Maximum test duration")
		dbURL      = flag.String("db", "", "Database URL for storing results (optional)")
		output     = flag.String("output", "console", "Output format: console, json, csv")
	)
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configFile, *baseURL, *users, *requests, *duration)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup database connection if provided
	var db *sqlx.DB
	if *dbURL != "" {
		db, err = sqlx.Connect("postgres", *dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			defer db.Close()
			if err := createBenchmarkTable(db); err != nil {
				log.Printf("Warning: Failed to create benchmark table: %v", err)
			}
		}
	}

	// Create benchmark suite
	suite := benchmark.NewBenchmarkSuite(config, db)

	// Run the benchmark
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	fmt.Printf("Starting benchmark test: %s\n", *testName)
	fmt.Printf("Target URL: %s\n", *baseURL)
	fmt.Printf("Configuration: %d users, %d requests per user\n", *users, *requests)
	fmt.Println("---")

	result, err := suite.RunLoadTest(ctx, *testName)
	if err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Save results to database if available
	if db != nil {
		if err := suite.SaveResult(result); err != nil {
			log.Printf("Warning: Failed to save results to database: %v", err)
		} else {
			fmt.Println("Results saved to database")
		}
	}

	// Output results
	switch *output {
	case "json":
		outputJSON(result)
	case "csv":
		outputCSV(result)
	default:
		outputConsole(result)
	}
}

func loadConfig(configFile, baseURL string, users, requests int, duration time.Duration) (*benchmark.BenchmarkConfig, error) {
	// Try to load from file first
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		var config benchmark.BenchmarkConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		return &config, nil
	}

	// Create default configuration
	config := &benchmark.BenchmarkConfig{
		BaseURL:         baseURL,
		ConcurrentUsers: users,
		RequestsPerUser: requests,
		TestDuration:    duration,
		Endpoints: []benchmark.EndpointConfig{
			{
				Name:    "list-articles",
				Method:  "GET",
				Path:    "/api/articles",
				Headers: map[string]string{"Accept": "application/json"},
				Weight:  50,
			},
			{
				Name:    "get-article",
				Method:  "GET",
				Path:    "/api/articles/1",
				Headers: map[string]string{"Accept": "application/json"},
				Weight:  30,
			},
			{
				Name:    "get-bias-analysis",
				Method:  "GET",
				Path:    "/api/articles/1/bias",
				Headers: map[string]string{"Accept": "application/json"},
				Weight:  20,
			},
		},
	}

	return config, nil
}

func createBenchmarkTable(db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS benchmark_results (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			total_requests INTEGER NOT NULL,
			successful_requests INTEGER NOT NULL,
			failed_requests INTEGER NOT NULL,
			average_latency_ms BIGINT NOT NULL,
			min_latency_ms BIGINT NOT NULL,
			max_latency_ms BIGINT NOT NULL,
			p95_latency_ms BIGINT NOT NULL,
			p99_latency_ms BIGINT NOT NULL,
			requests_per_second DECIMAL(10,2) NOT NULL,
			error_rate DECIMAL(5,2) NOT NULL,
			total_duration_ms BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

func outputConsole(result *benchmark.BenchmarkResult) {
	fmt.Println("\n=== BENCHMARK RESULTS ===")
	fmt.Printf("Test Name: %s\n", result.TestName)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("Total Duration: %v\n", result.TotalDuration)
	fmt.Println()

	fmt.Println("REQUEST STATISTICS:")
	fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("  Successful: %d\n", result.SuccessfulReqs)
	fmt.Printf("  Failed: %d\n", result.FailedRequests)
	fmt.Printf("  Error Rate: %.2f%%\n", result.ErrorRate)
	fmt.Printf("  Requests/sec: %.2f\n", result.RequestsPerSec)
	fmt.Println()

	fmt.Println("LATENCY STATISTICS:")
	fmt.Printf("  Average: %v\n", result.AverageLatency)
	fmt.Printf("  Minimum: %v\n", result.MinLatency)
	fmt.Printf("  Maximum: %v\n", result.MaxLatency)
	fmt.Printf("  95th Percentile: %v\n", result.P95Latency)
	fmt.Printf("  99th Percentile: %v\n", result.P99Latency)
	fmt.Println()
}

func outputJSON(result *benchmark.BenchmarkResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

func outputCSV(result *benchmark.BenchmarkResult) {
	fmt.Println("test_name,timestamp,total_requests,successful_requests,failed_requests,average_latency_ms,min_latency_ms,max_latency_ms,p95_latency_ms,p99_latency_ms,requests_per_second,error_rate,total_duration_ms")
	fmt.Printf("%s,%s,%d,%d,%d,%d,%d,%d,%d,%d,%.2f,%.2f,%d\n",
		result.TestName,
		result.Timestamp.Format(time.RFC3339),
		result.TotalRequests,
		result.SuccessfulReqs,
		result.FailedRequests,
		result.AverageLatency.Milliseconds(),
		result.MinLatency.Milliseconds(),
		result.MaxLatency.Milliseconds(),
		result.P95Latency.Milliseconds(),
		result.P99Latency.Milliseconds(),
		result.RequestsPerSec,
		result.ErrorRate,
		result.TotalDuration.Milliseconds(),
	)
}
