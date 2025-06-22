package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	internaltesting "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// TestAPIIntegration demonstrates API testing with test server management
func TestAPIIntegration(t *testing.T) { // Setup test database
	dbConfig := internaltesting.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
		MigrationsPath: "../migrations",
		SeedDataPath:   "../testdata/seed",
	}

	testDB := internaltesting.SetupTestDatabase(t, dbConfig)
	defer testDB.Cleanup()

	// Setup test server
	serverConfig := internaltesting.DefaultTestServerConfig()
	serverConfig.Environment["DB_CONNECTION"] = testDB.GetConnectionString()
	serverConfig.Environment["TEST_MODE"] = "true"

	serverManager := internaltesting.NewTestServerManager(serverConfig)
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

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
		ExpectedStatus: http.StatusOK,
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
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("Failed to create test article, status: %d", resp.StatusCode)
			}
		},
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var reanalyzeResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&reanalyzeResponse); err != nil {
				t.Fatalf("Failed to decode reanalyze response: %v", err)
			}

			// Check for success response
			if success, ok := reanalyzeResponse["success"].(bool); !ok || !success {
				t.Errorf("Expected successful reanalysis, got response: %v", reanalyzeResponse)
			}
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
		// In CI with NO_AUTO_ANALYZE=true, ensemble data won't exist, so expect 404
		// In normal environments, expect 200
		ExpectedStatus: func() int {
			if os.Getenv("NO_AUTO_ANALYZE") == "true" {
				return http.StatusNotFound
			}
			return http.StatusOK
		}(),
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
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("Failed to create test article, status: %d", resp.StatusCode)
			}
		},
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var ensembleResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&ensembleResponse); err != nil {
				t.Fatalf("Failed to decode ensemble response: %v", err)
			}

			// In CI environment, expect 404 with error message
			if os.Getenv("NO_AUTO_ANALYZE") == "true" {
				if resp.StatusCode == http.StatusNotFound {
					// Validate error response structure
					if success, ok := ensembleResponse["success"].(bool); !ok || success {
						t.Errorf("Expected error response in CI environment, got: %v", ensembleResponse)
					}
					if errorData, ok := ensembleResponse["error"].(map[string]interface{}); ok {
						if code, ok := errorData["code"].(string); !ok || code != "not_found" {
							t.Errorf("Expected 'not_found' error code, got: %v", code)
						}
					}
					t.Logf("✅ CI Environment: Correctly returned 404 for ensemble data (expected behavior)")
					return
				}
			}

			// In normal environment, expect success response
			if success, ok := ensembleResponse["success"].(bool); !ok || !success {
				t.Errorf("Expected successful ensemble response, got: %v", ensembleResponse)
			}
		},
	})

	// Run all test cases
	suite.RunTests(t)
}

// TestAPIPerformance demonstrates performance testing
func TestAPIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	// Setup test server
	serverConfig := internaltesting.DefaultTestServerConfig()
	serverManager := internaltesting.NewTestServerManager(serverConfig)
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
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
