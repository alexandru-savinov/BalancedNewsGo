package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// BenchmarkResult represents the results of a performance benchmark
type BenchmarkResult struct {
	TestName        string        `json:"test_name"`
	Timestamp       time.Time     `json:"timestamp"`
	TotalRequests   int           `json:"total_requests"`
	SuccessfulReqs  int           `json:"successful_requests"`
	FailedRequests  int           `json:"failed_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	RequestsPerSec  float64       `json:"requests_per_second"`
	ErrorRate       float64       `json:"error_rate"`
	TotalDuration   time.Duration `json:"total_duration"`
}

// BenchmarkConfig holds configuration for benchmark tests
type BenchmarkConfig struct {
	BaseURL         string
	ConcurrentUsers int
	RequestsPerUser int
	TestDuration    time.Duration
	Endpoints       []EndpointConfig
}

// EndpointConfig defines an endpoint to benchmark
type EndpointConfig struct {
	Name     string            `json:"name"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers"`
	Body     interface{}       `json:"body,omitempty"`
	Weight   int               `json:"weight"` // Relative frequency of this endpoint
}

// RequestResult holds the result of a single HTTP request
type RequestResult struct {
	Success   bool
	Latency   time.Duration
	Status    int
	Error     error
	Timestamp time.Time
}

// BenchmarkSuite manages and runs performance benchmarks
type BenchmarkSuite struct {
	config *BenchmarkConfig
	client *http.Client
	db     *sqlx.DB
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(config *BenchmarkConfig, db *sqlx.DB) *BenchmarkSuite {
	return &BenchmarkSuite{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		db: db,
	}
}

// RunLoadTest executes a load test with the configured parameters
func (bs *BenchmarkSuite) RunLoadTest(ctx context.Context, testName string) (*BenchmarkResult, error) {
	fmt.Printf("Starting load test: %s\n", testName)
	fmt.Printf("Concurrent users: %d, Requests per user: %d\n", bs.config.ConcurrentUsers, bs.config.RequestsPerUser)

	startTime := time.Now()
	results := make(chan RequestResult, bs.config.ConcurrentUsers*bs.config.RequestsPerUser)
	
	var wg sync.WaitGroup
	
	// Start concurrent workers
	for i := 0; i < bs.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			bs.runUserSession(ctx, userID, results)
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []RequestResult
	for result := range results {
		allResults = append(allResults, result)
	}

	endTime := time.Now()
	
	// Calculate statistics
	return bs.calculateStats(testName, allResults, startTime, endTime), nil
}

// runUserSession simulates a single user's session
func (bs *BenchmarkSuite) runUserSession(ctx context.Context, userID int, results chan<- RequestResult) {
	for i := 0; i < bs.config.RequestsPerUser; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			endpoint := bs.selectEndpoint()
			result := bs.makeRequest(ctx, endpoint)
			results <- result
			
			// Small delay between requests to simulate real user behavior
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// selectEndpoint selects an endpoint based on weights
func (bs *BenchmarkSuite) selectEndpoint() EndpointConfig {
	if len(bs.config.Endpoints) == 0 {
		return EndpointConfig{
			Name:   "default",
			Method: "GET",
			Path:   "/api/articles",
		}
	}
	
	// Simple round-robin for now, could be enhanced with weighted selection
	return bs.config.Endpoints[0]
}

// makeRequest executes a single HTTP request
func (bs *BenchmarkSuite) makeRequest(ctx context.Context, endpoint EndpointConfig) RequestResult {
	startTime := time.Now()
	
	var body io.Reader
	if endpoint.Body != nil {
		jsonBody, err := json.Marshal(endpoint.Body)
		if err != nil {
			return RequestResult{
				Success:   false,
				Latency:   time.Since(startTime),
				Error:     err,
				Timestamp: startTime,
			}
		}
		body = bytes.NewReader(jsonBody)
	}

	url := bs.config.BaseURL + endpoint.Path
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, url, body)
	if err != nil {
		return RequestResult{
			Success:   false,
			Latency:   time.Since(startTime),
			Error:     err,
			Timestamp: startTime,
		}
	}

	// Set headers
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}
	
	if endpoint.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := bs.client.Do(req)
	latency := time.Since(startTime)
	
	if err != nil {
		return RequestResult{
			Success:   false,
			Latency:   latency,
			Error:     err,
			Timestamp: startTime,
		}
	}
	defer resp.Body.Close()

	// Read response body to ensure complete request
	_, _ = io.ReadAll(resp.Body)

	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	
	return RequestResult{
		Success:   success,
		Latency:   latency,
		Status:    resp.StatusCode,
		Timestamp: startTime,
	}
}

// calculateStats computes benchmark statistics from request results
func (bs *BenchmarkSuite) calculateStats(testName string, results []RequestResult, startTime, endTime time.Time) *BenchmarkResult {
	if len(results) == 0 {
		return &BenchmarkResult{
			TestName:  testName,
			Timestamp: startTime,
		}
	}

	totalRequests := len(results)
	successfulReqs := 0
	var latencies []time.Duration
	var totalLatency time.Duration

	for _, result := range results {
		if result.Success {
			successfulReqs++
		}
		latencies = append(latencies, result.Latency)
		totalLatency += result.Latency
	}

	// Sort latencies for percentile calculations
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	failedRequests := totalRequests - successfulReqs
	errorRate := float64(failedRequests) / float64(totalRequests) * 100
	totalDuration := endTime.Sub(startTime)
	requestsPerSec := float64(totalRequests) / totalDuration.Seconds()

	result := &BenchmarkResult{
		TestName:        testName,
		Timestamp:       startTime,
		TotalRequests:   totalRequests,
		SuccessfulReqs:  successfulReqs,
		FailedRequests:  failedRequests,
		AverageLatency:  totalLatency / time.Duration(totalRequests),
		MinLatency:      latencies[0],
		MaxLatency:      latencies[len(latencies)-1],
		P95Latency:      latencies[int(float64(len(latencies))*0.95)],
		P99Latency:      latencies[int(float64(len(latencies))*0.99)],
		RequestsPerSec:  requestsPerSec,
		ErrorRate:       errorRate,
		TotalDuration:   totalDuration,
	}

	return result
}

// SaveResult saves benchmark results to the database
func (bs *BenchmarkSuite) SaveResult(result *BenchmarkResult) error {
	if bs.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `
		INSERT INTO benchmark_results (
			test_name, timestamp, total_requests, successful_requests, failed_requests,
			average_latency_ms, min_latency_ms, max_latency_ms, p95_latency_ms, p99_latency_ms,
			requests_per_second, error_rate, total_duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := bs.db.Exec(query,
		result.TestName,
		result.Timestamp,
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

	return err
}
