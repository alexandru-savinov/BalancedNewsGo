package testing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// APITestCase represents a single API test case
type APITestCase struct {
	Name           string
	Method         string
	Path           string
	Headers        map[string]string
	Body           interface{}
	ExpectedStatus int
	ExpectedBody   interface{}
	Setup          func(*testing.T)
	Cleanup        func(*testing.T)
	ValidateFunc   func(*testing.T, *http.Response)
}

// APITestSuite manages a collection of API tests
type APITestSuite struct {
	BaseURL       string
	Client        *http.Client
	TestCases     []APITestCase
	CommonSetup   func(*testing.T)
	CommonCleanup func(*testing.T)
}

// NewAPITestSuite creates a new API test suite
func NewAPITestSuite(baseURL string) *APITestSuite {
	// Create HTTP client with disabled keep-alive for test environments
	transport := &http.Transport{}

	// Disable keep-alive in test environments to prevent hanging processes
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" || os.Getenv("CI") == "true" {
		transport.DisableKeepAlives = true
		transport.MaxIdleConnsPerHost = 0
		transport.IdleConnTimeout = 1 * time.Second
	}

	return &APITestSuite{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		TestCases: make([]APITestCase, 0),
	}
}

// AddTestCase adds a test case to the suite
func (suite *APITestSuite) AddTestCase(testCase APITestCase) {
	suite.TestCases = append(suite.TestCases, testCase)
}

// Cleanup forces cleanup of HTTP connections to prevent I/O timeouts
func (suite *APITestSuite) Cleanup() {
	if suite.Client != nil && suite.Client.Transport != nil {
		if transport, ok := suite.Client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
}

// RunTests executes all test cases in the suite
func (suite *APITestSuite) RunTests(t *testing.T) {
	for _, testCase := range suite.TestCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Run common setup
			if suite.CommonSetup != nil {
				suite.CommonSetup(t)
			}

			// Run test-specific setup
			if testCase.Setup != nil {
				testCase.Setup(t)
			}

			// Execute the test
			suite.executeTestCase(t, testCase)

			// Run test-specific cleanup
			if testCase.Cleanup != nil {
				testCase.Cleanup(t)
			}

			// Run common cleanup
			if suite.CommonCleanup != nil {
				suite.CommonCleanup(t)
			}
		})
	}
}

// executeTestCase executes a single test case
func (suite *APITestSuite) executeTestCase(t *testing.T, testCase APITestCase) {
	t.Helper()

	// Create request
	req, err := suite.createRequest(testCase)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Execute request
	resp, err := suite.Client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Validate status code
	if resp.StatusCode != testCase.ExpectedStatus {
		t.Errorf("Expected status %d, got %d", testCase.ExpectedStatus, resp.StatusCode)
	}

	// Run custom validation if provided
	if testCase.ValidateFunc != nil {
		testCase.ValidateFunc(t, resp)
	}
}

// createRequest creates an HTTP request from a test case
func (suite *APITestSuite) createRequest(testCase APITestCase) (*http.Request, error) {
	url := suite.BaseURL + testCase.Path

	var body strings.Reader
	if testCase.Body != nil {
		bodyBytes, err := json.Marshal(testCase.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = *strings.NewReader(string(bodyBytes))
	}

	var req *http.Request
	var err error

	if testCase.Body != nil {
		req, err = http.NewRequest(testCase.Method, url, &body)
	} else {
		req, err = http.NewRequest(testCase.Method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range testCase.Headers {
		req.Header.Set(key, value)
	}

	// Set default content type for JSON requests
	if testCase.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// MockHandler creates a mock HTTP handler for testing
type MockHandler struct {
	responses map[string]*http.Response
	requests  []*http.Request
}

// NewMockHandler creates a new mock handler
func NewMockHandler() *MockHandler {
	return &MockHandler{
		responses: make(map[string]*http.Response),
		requests:  make([]*http.Request, 0),
	}
}

// AddResponse adds a mock response for a specific path
func (mh *MockHandler) AddResponse(path string, statusCode int, body interface{}) {
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(statusCode)

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		recorder.Write(bodyBytes)
	}

	mh.responses[path] = recorder.Result()
}

// ServeHTTP implements the http.Handler interface
func (mh *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Store the request for verification
	mh.requests = append(mh.requests, r)

	// Find matching response
	if resp, exists := mh.responses[r.URL.Path]; exists {
		// Copy headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Set status code
		w.WriteHeader(resp.StatusCode)

		// Copy body
		if resp.Body != nil {
			defer resp.Body.Close()
			bodyBytes := make([]byte, 1024)
			n, _ := resp.Body.Read(bodyBytes)
			w.Write(bodyBytes[:n])
		}
	} else {
		// Default 404 response
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}
}

// GetRequests returns all captured requests
func (mh *MockHandler) GetRequests() []*http.Request {
	return mh.requests
}

// ClearRequests clears the captured requests
func (mh *MockHandler) ClearRequests() {
	mh.requests = make([]*http.Request, 0)
}

// PerformanceTestConfig holds configuration for performance testing
type PerformanceTestConfig struct {
	URL               string
	Method            string
	Headers           map[string]string
	Body              interface{}
	ConcurrentUsers   int
	RequestsPerUser   int
	TestDuration      time.Duration
	AcceptableLatency time.Duration
}

// PerformanceTestResult holds the results of a performance test
type PerformanceTestResult struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	RequestsPerSecond  float64
	ErrorRate          float64
}

// RunPerformanceTest executes a performance test
func RunPerformanceTest(t *testing.T, config PerformanceTestConfig) *PerformanceTestResult {
	t.Helper()

	// Implementation would go here for performance testing
	// This is a placeholder structure

	return &PerformanceTestResult{
		TotalRequests:      config.ConcurrentUsers * config.RequestsPerUser,
		SuccessfulRequests: config.ConcurrentUsers * config.RequestsPerUser,
		FailedRequests:     0,
		AverageLatency:     100 * time.Millisecond,
		MaxLatency:         200 * time.Millisecond,
		MinLatency:         50 * time.Millisecond,
		RequestsPerSecond:  100.0,
		ErrorRate:          0.0,
	}
}
