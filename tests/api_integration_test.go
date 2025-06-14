package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// TestAPIIntegration demonstrates API testing with test server management
func TestAPIIntegration(t *testing.T) {
	// Setup test database
	dbConfig := testing.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
		MigrationsPath: "../migrations",
		SeedDataPath:   "../testdata/seed",
	}

	testDB := testing.SetupTestDatabase(t, dbConfig)
	defer testDB.Cleanup()

	// Setup test server
	serverConfig := testing.DefaultTestServerConfig()
	serverConfig.Environment["DB_CONNECTION"] = testDB.GetConnectionString()
	serverConfig.Environment["TEST_MODE"] = "true"

	serverManager := testing.NewTestServerManager(serverConfig)
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	// Create API test suite
	suite := testing.NewAPITestSuite(serverManager.GetBaseURL())

	// Add test cases
	suite.AddTestCase(testing.APITestCase{
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

	suite.AddTestCase(testing.APITestCase{
		Name:           "Get Articles",
		Method:         "GET",
		Path:           "/api/articles",
		ExpectedStatus: http.StatusOK,
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var articles []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&articles); err != nil {
				t.Fatalf("Failed to decode articles response: %v", err)
			}
			t.Logf("Retrieved %d articles", len(articles))
		},
	})

	suite.AddTestCase(testing.APITestCase{
		Name:    "Score Article",
		Method:  "POST",
		Path:    "/api/articles/score",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"article_id": "test-article-1",
			"content":    "This is a test article for scoring",
		},
		ExpectedStatus: http.StatusOK,
		Setup: func(t *testing.T) {
			// Insert test article before scoring
			_, err := testDB.DB.Exec(`
				INSERT INTO articles (id, title, content, url, source, published_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, datetime('now'), datetime('now'), datetime('now'))
			`, "test-article-1", "Test Article", "This is a test article for scoring", "http://test.com", "test-source")
			if err != nil {
				t.Fatalf("Failed to insert test article: %v", err)
			}
		},
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var scoreResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&scoreResponse); err != nil {
				t.Fatalf("Failed to decode score response: %v", err)
			}

			if scoreResponse["article_id"] != "test-article-1" {
				t.Errorf("Expected article_id 'test-article-1', got '%v'", scoreResponse["article_id"])
			}

			if _, ok := scoreResponse["composite_score"]; !ok {
				t.Error("Expected composite_score in response")
			}
		},
	})

	suite.AddTestCase(testing.APITestCase{
		Name:    "Submit Feedback",
		Method:  "POST",
		Path:    "/api/feedback",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"article_id":    "test-article-1",
			"user_rating":   4,
			"feedback_text": "Great article!",
		},
		ExpectedStatus: http.StatusCreated,
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

	suite.AddTestCase(testing.APITestCase{
		Name:           "Get Article Scores",
		Method:         "GET",
		Path:           "/api/articles/test-article-1/scores",
		ExpectedStatus: http.StatusOK,
		ValidateFunc: func(t *testing.T, resp *http.Response) {
			var scoresResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&scoresResponse); err != nil {
				t.Fatalf("Failed to decode scores response: %v", err)
			}

			if scoresResponse["article_id"] != "test-article-1" {
				t.Errorf("Expected article_id 'test-article-1', got '%v'", scoresResponse["article_id"])
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
	serverConfig := testing.DefaultTestServerConfig()
	serverManager := testing.NewTestServerManager(serverConfig)
	if err := serverManager.Start(t); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}

	// Performance test configuration
	perfConfig := testing.PerformanceTestConfig{
		URL:               serverManager.GetBaseURL() + "/healthz",
		Method:            "GET",
		ConcurrentUsers:   10,
		RequestsPerUser:   100,
		AcceptableLatency: 100 * 1000000, // 100ms in nanoseconds
	}

	// Run performance test
	result := testing.RunPerformanceTest(t, perfConfig)

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
