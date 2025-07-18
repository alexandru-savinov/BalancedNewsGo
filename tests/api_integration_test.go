package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	internaltesting "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// TestAPIIntegration demonstrates API testing with test server management
func TestAPIIntegration(t *testing.T) {
	t.Logf("🔍 DEBUG: TestAPIIntegration starting")

	// Set test timeout to prevent CI/CD hanging
	if deadline, ok := t.Deadline(); ok {
		t.Logf("Test deadline: %v", deadline)
	}

	t.Logf("🔍 DEBUG: Setting up test database")
	// Setup test database
	dbConfig := internaltesting.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
		MigrationsPath: "../migrations",
		SeedDataPath:   "../testdata/seed",
	}

	testDB := internaltesting.SetupTestDatabase(t, dbConfig)
	defer func() {
		if err := testDB.Cleanup(); err != nil {
			t.Logf("Failed to cleanup test database: %v", err)
		}
	}()

	// Setup test server
	serverConfig := internaltesting.DefaultTestServerConfig()
	serverConfig.Environment["DB_CONNECTION"] = testDB.GetConnectionString()
	serverConfig.Environment["TEST_MODE"] = "true"

	t.Logf("🔍 DEBUG: Creating test server manager")
	serverManager := internaltesting.NewTestServerManager(serverConfig)

	t.Logf("🔍 DEBUG: Starting test server")
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	t.Logf("🔍 DEBUG: Test server started successfully")

	// Ensure explicit cleanup for CI/CD environments
	defer func() {
		t.Logf("🔍 DEBUG: Starting comprehensive cleanup")

		// Stop the server first
		if err := serverManager.Stop(); err != nil {
			t.Logf("Warning: Failed to stop server cleanly: %v", err)
		}
		t.Logf("🔍 DEBUG: Server stopped")

		// Force garbage collection to clean up any remaining resources
		runtime.GC()
		t.Logf("🔍 DEBUG: Garbage collection completed")

		// Minimal sleep to allow cleanup
		time.Sleep(50 * time.Millisecond)
		t.Logf("🔍 DEBUG: Final cleanup completed")
	}()

	// Create API test suite
	suite := internaltesting.NewAPITestSuite(serverManager.GetBaseURL())

	// Add test cases
	suite.AddTestCase(internaltesting.APITestCase{
		Name:           "Health Check",
		Method:         "GET",
		Path:           "/healthz",
		ExpectedStatus: http.StatusOK,
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if response["status"] != "ok" {
				t.Errorf("Expected status 'ok', got '%v'", response["status"])
			}
		},
	})

	suite.AddTestCase(internaltesting.APITestCase{
		Name:           "Get Articles",
		Method:         "GET",
		Path:           "/api/articles",
		ExpectedStatus: http.StatusOK,
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode articles response: %v", err)
			}
			// Check if response has success field and data array
			if success, ok := response["success"].(bool); ok && success {
				if data, ok := response["data"].([]interface{}); ok {
					t.Logf("Retrieved %d articles", len(data))
				} else {
					t.Logf("Response data is not an array: %v", response["data"])
				}
			} else {
				t.Logf("Response: %v", response)
			}
		},
	})

	suite.AddTestCase(internaltesting.APITestCase{
		Name:    "Reanalyze Article",
		Method:  "POST",
		Path:    "/api/llm/reanalyze/1",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"force": true,
		},
		// Dynamic status based on environment:
		// - 200 in CI with NO_AUTO_ANALYZE=true (skips LLM analysis)
		// - 503 in local environments without valid API keys
		ExpectedStatus: func() int {
			if os.Getenv("NO_AUTO_ANALYZE") == "true" {
				return http.StatusOK
			}
			return http.StatusServiceUnavailable
		}(),
		Setup: func(t *testing.T) {
			// Create test article via API with unique URL
			timestamp := time.Now().UnixNano()
			articleData := map[string]interface{}{
				"title":    "Test Article",
				"content":  "This is a test article for scoring",
				"url":      fmt.Sprintf("https://test.com/article1-%d", timestamp),
				"source":   "test-source",
				"pub_date": "2025-06-21T12:00:00Z",
			}

			bodyBytes, _ := json.Marshal(articleData)
			resp, err := http.Post(serverManager.GetBaseURL()+"/api/articles", "application/json", strings.NewReader(string(bodyBytes)))
			if err != nil {
				t.Fatalf("Failed to create test article: %v", err)
			}
			defer func() {
				if closeErr := resp.Body.Close(); closeErr != nil {
					t.Logf("Warning: failed to close response body: %v", closeErr)
				}
			}()

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("Failed to create test article, status: %d", resp.StatusCode)
			}
		},
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var reanalyzeResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&reanalyzeResponse); err != nil {
				t.Fatalf("Failed to decode reanalyze response: %v", err)
			}

			// Expect 200 in CI with NO_AUTO_ANALYZE=true (skips LLM analysis but returns success)
			if resp.StatusCode == http.StatusOK {
				// Validate success response structure
				if success, ok := reanalyzeResponse["success"].(bool); !ok || !success {
					t.Errorf("Expected successful reanalysis response, got: %v", reanalyzeResponse)
				}
				t.Logf("✅ CI Environment: Correctly returned 200 for reanalysis with NO_AUTO_ANALYZE=true (expected behavior)")
				return
			}

			// Handle 503 case for local environments without API keys
			if resp.StatusCode == http.StatusServiceUnavailable {
				// Validate error response structure
				if success, ok := reanalyzeResponse["success"].(bool); !ok || success {
					t.Errorf("Expected error response for 503, got: %v", reanalyzeResponse)
				}
				if errorData, ok := reanalyzeResponse["error"].(map[string]interface{}); ok {
					if code, ok := errorData["code"].(string); !ok || code != "llm_service_error" {
						t.Errorf("Expected 'llm_service_error' error code, got: %v", code)
					}
				}
				t.Logf("✅ Local Environment: Correctly returned 503 for LLM service unavailable (expected behavior without valid API keys)")
				return
			}

			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		},
	})

	suite.AddTestCase(internaltesting.APITestCase{
		Name:    "Submit Feedback",
		Method:  "POST",
		Path:    "/api/feedback",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"article_id":    1,
			"user_id":       "test-user",
			"feedback_text": "Great article!",
			"category":      "agree",
		},
		ExpectedStatus: http.StatusOK,
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var feedbackResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&feedbackResponse); err != nil {
				t.Fatalf("Failed to decode feedback response: %v", err)
			}

			if feedbackResponse["success"] != true {
				t.Error("Expected successful feedback submission")
			}
		},
	})

	suite.AddTestCase(internaltesting.APITestCase{
		Name:   "Get Article Ensemble Details",
		Method: "GET",
		Path:   "/api/articles/1/ensemble",
		// Expect 404 in most test environments since ensemble data requires working LLM integration
		// This includes: CI environments, local development without API keys, etc.
		ExpectedStatus: http.StatusNotFound,
		Setup: func(t *testing.T) {
			// Create test article via API for ensemble details with unique URL
			timestamp := time.Now().UnixNano()
			articleData := map[string]interface{}{
				"title":    "Test Article",
				"content":  "This is a test article for ensemble",
				"url":      fmt.Sprintf("https://test.com/article2-%d", timestamp),
				"source":   "test-source",
				"pub_date": "2025-06-21T12:00:00Z",
			}

			bodyBytes, _ := json.Marshal(articleData)
			resp, err := http.Post(serverManager.GetBaseURL()+"/api/articles", "application/json", strings.NewReader(string(bodyBytes)))
			if err != nil {
				t.Fatalf("Failed to create test article: %v", err)
			}
			defer func() {
				if closeErr := resp.Body.Close(); closeErr != nil {
					t.Logf("Warning: failed to close response body: %v", closeErr)
				}
			}()

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("Failed to create test article, status: %d", resp.StatusCode)
			}
		},
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var ensembleResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&ensembleResponse); err != nil {
				t.Fatalf("Failed to decode ensemble response: %v", err)
			}

			// Expect 404 with proper error response structure
			if resp.StatusCode == http.StatusNotFound {
				// Validate error response structure
				if success, ok := ensembleResponse["success"].(bool); !ok || success {
					t.Errorf("Expected error response for 404, got: %v", ensembleResponse)
				}
				if errorData, ok := ensembleResponse["error"].(map[string]interface{}); ok {
					if code, ok := errorData["code"].(string); !ok || code != "not_found" {
						t.Errorf("Expected 'not_found' error code, got: %v", code)
					}
				}
				t.Logf("✅ Test Environment: Correctly returned 404 for ensemble data (expected behavior without LLM integration)")
				return
			}

			// If somehow we get 200 (with working LLM), validate success response
			if success, ok := ensembleResponse["success"].(bool); !ok || !success {
				t.Errorf("Expected successful ensemble response, got: %v", ensembleResponse)
			}
		},
	})

	// Run all test cases
	t.Logf("🔍 DEBUG: About to run test suite with %d test cases", len(suite.TestCases))
	suite.RunTests(t)
	t.Logf("🔍 DEBUG: Test suite completed successfully")

	// Force cleanup of HTTP connections
	t.Logf("🔍 DEBUG: Cleaning up HTTP connections")
	suite.Cleanup()
	t.Logf("🔍 DEBUG: HTTP connections cleaned up")
}

// TestAPIPerformance demonstrates performance testing
func TestAPIPerformance(t *testing.T) {
	t.Logf("🔍 DEBUG: TestAPIPerformance starting")
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	t.Logf("🔍 DEBUG: Setting up performance test server")
	// Setup test server
	serverConfig := internaltesting.DefaultTestServerConfig()
	serverManager := internaltesting.NewTestServerManager(serverConfig)
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	// Ensure explicit cleanup for CI/CD environments
	defer func() {
		t.Logf("🔍 DEBUG: Starting performance test cleanup")
		if err := serverManager.Stop(); err != nil {
			t.Logf("Warning: Failed to stop server cleanly: %v", err)
		}
		runtime.GC()
		t.Logf("🔍 DEBUG: Performance test cleanup completed")
	}()
	// Performance test configuration
	perfConfig := internaltesting.PerformanceTestConfig{
		URL:               serverManager.GetBaseURL() + "/healthz",
		Method:            "GET",
		ConcurrentUsers:   10,
		RequestsPerUser:   100,
		AcceptableLatency: 100 * 1000000, // 100ms in nanoseconds
	}

	// Run performance test
	result := internaltesting.RunPerformanceTest(t, perfConfig)

	// Validate performance metrics
	if result.ErrorRate > 0.01 { // Allow up to 1% error rate
		t.Errorf("Error rate too high: %.2f%%", result.ErrorRate*100)
	}

	if result.AverageLatency > perfConfig.AcceptableLatency {
		t.Errorf("Average latency too high: %v (acceptable: %v)",
			result.AverageLatency, perfConfig.AcceptableLatency)
	}

	if result.RequestsPerSecond < 50 { // Minimum 50 RPS
		t.Errorf("Requests per second too low: %.2f", result.RequestsPerSecond)
	}

	t.Logf("Performance test results:")
	t.Logf("  Total requests: %d", result.TotalRequests)
	t.Logf("  Successful requests: %d", result.SuccessfulRequests)
	t.Logf("  Failed requests: %d", result.FailedRequests)
	t.Logf("  Average latency: %v", result.AverageLatency)
	t.Logf("  Requests per second: %.2f", result.RequestsPerSecond)
	t.Logf("  Error rate: %.2f%%", result.ErrorRate*100)
}
