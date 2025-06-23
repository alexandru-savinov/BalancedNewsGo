package testing

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestServerConfig holds configuration for test server management
type TestServerConfig struct {
	Port            int
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	HealthEndpoint  string
	ServerCommand   []string
	Environment     map[string]string
}

// DefaultTestServerConfig returns a default server configuration
func DefaultTestServerConfig() TestServerConfig {
	// Use OS-appropriate temp directory
	logPath := os.TempDir() + "/test_server.log"
	env := map[string]string{
		"TEST_MODE":             "true",
		"LOG_FILE_PATH":         logPath,
		"GIN_MODE":              "test",
		"DB_CONNECTION":         ":memory:", // Use in-memory SQLite for tests
		"PORT":                  "8080",
		"LLM_API_KEY":           "test-key",           // Required for LLM client initialization
		"LLM_API_KEY_SECONDARY": "test-secondary-key", // Required for LLM client initialization
		"LLM_BASE_URL":          "https://openrouter.ai/api/v1/chat/completions",
	}
	// Use a random port to avoid conflicts with other test runs
	port := 8080
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
		// Use a different port range for tests to avoid conflicts
		port = 8090 + (os.Getpid() % 10) // Use PID to get a unique port
	}

	// Add PORT environment variable to the test environment
	env["PORT"] = fmt.Sprintf("%d", port)

	// Use compiled binary in CI environments to avoid I/O timeout issues
	serverCommand := []string{"go", "run", "./cmd/server"}
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		// In CI, build a temporary binary to avoid go run I/O issues
		serverCommand = []string{"./test-server-ci"}
	} else if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
		// Check if test-server.exe exists, use it to avoid build conflicts
		if _, err := os.Stat("test-server.exe"); err == nil {
			serverCommand = []string{"./test-server.exe"}
		}
	}

	return TestServerConfig{
		Port:            port,
		StartupTimeout:  30 * time.Second, // Reduced timeout for faster CI
		ShutdownTimeout: 5 * time.Second,  // Reduced shutdown timeout
		HealthEndpoint:  "/healthz",
		ServerCommand:   serverCommand,
		Environment:     env,
	}
}

// TestServerManager manages a test server instance
type TestServerManager struct {
	config TestServerConfig
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// NewTestServerManager creates a new test server manager
func NewTestServerManager(config TestServerConfig) *TestServerManager {
	return &TestServerManager{
		config: config,
	}
}

// Start starts the test server
func (tsm *TestServerManager) Start(t *testing.T) error {
	t.Helper()

	startTime := time.Now()
	t.Logf("🚀 [%v] TestServerManager.Start() beginning", startTime.Format("15:04:05.000"))

	// Check if server is already running
	t.Logf("🔍 [%v] Checking if server is already healthy on port %d", time.Now().Format("15:04:05.000"), tsm.config.Port)
	if tsm.IsHealthy() {
		t.Logf("✅ [%v] Server already running and healthy", time.Now().Format("15:04:05.000"))
		return nil
	}
	t.Logf("📝 [%v] Server not healthy, proceeding with startup", time.Now().Format("15:04:05.000"))

	// Create context for server process
	t.Logf("🔧 [%v] Creating context for server process", time.Now().Format("15:04:05.000"))
	ctx, cancel := context.WithCancel(context.Background())
	tsm.cancel = cancel

	// Setup server command
	t.Logf("🔧 [%v] Setting up server command: %v", time.Now().Format("15:04:05.000"), tsm.config.ServerCommand)
	tsm.cmd = exec.CommandContext(ctx, tsm.config.ServerCommand[0], tsm.config.ServerCommand[1:]...)

	// Set working directory to project root
	// Try to detect if we're in tests directory and go up, otherwise use current directory
	workingDir := "."
	t.Logf("🔍 [%v] Detecting working directory...", time.Now().Format("15:04:05.000"))
	if _, err := os.Stat("../cmd/server"); err == nil {
		workingDir = ".."
		t.Logf("📁 [%v] Found ../cmd/server, using parent directory: %s", time.Now().Format("15:04:05.000"), workingDir)
	} else {
		t.Logf("📁 [%v] Using current directory: %s (../cmd/server not found: %v)", time.Now().Format("15:04:05.000"), workingDir, err)
	}
	tsm.cmd.Dir = workingDir

	// Copy current environment and add test-specific variables
	t.Logf("🌍 [%v] Setting up environment variables", time.Now().Format("15:04:05.000"))
	tsm.cmd.Env = os.Environ()
	for key, value := range tsm.config.Environment {
		tsm.cmd.Env = append(tsm.cmd.Env, fmt.Sprintf("%s=%s", key, value))
		t.Logf("🔧 [%v] Set env var: %s=%s", time.Now().Format("15:04:05.000"), key, value)
	}

	// Capture server output for debugging - ALWAYS capture in CI for debugging
	t.Logf("🔧 [%v] Configuring server output capture", time.Now().Format("15:04:05.000"))
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		// In CI environments, capture output for debugging server startup issues
		t.Logf("🔍 [%v] CI environment detected - capturing server output for debugging", time.Now().Format("15:04:05.000"))
		tsm.cmd.Stdout = os.Stdout
		tsm.cmd.Stderr = os.Stderr
	} else if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
		// In test environments, discard output to prevent I/O timeout
		t.Logf("🔇 [%v] Test mode - discarding server output", time.Now().Format("15:04:05.000"))
		tsm.cmd.Stdout = nil
		tsm.cmd.Stderr = nil
	} else {
		// In normal environments, show output
		t.Logf("📺 [%v] Normal mode - showing server output", time.Now().Format("15:04:05.000"))
		tsm.cmd.Stdout = os.Stdout
		tsm.cmd.Stderr = os.Stderr
	}

	// Start the server
	t.Logf("🚀 [%v] Starting server process...", time.Now().Format("15:04:05.000"))
	if err := tsm.cmd.Start(); err != nil {
		t.Logf("❌ [%v] Failed to start server process: %v", time.Now().Format("15:04:05.000"), err)
		cancel()
		return fmt.Errorf("failed to start server: %w", err)
	}
	t.Logf("✅ [%v] Server process started successfully (PID: %d)", time.Now().Format("15:04:05.000"), tsm.cmd.Process.Pid)

	// Wait for server to be healthy
	t.Logf("⏳ [%v] Waiting for server to become healthy (timeout: %v)", time.Now().Format("15:04:05.000"), tsm.config.StartupTimeout)
	if err := tsm.waitForHealth(); err != nil {
		t.Logf("❌ [%v] Server failed to become healthy: %v", time.Now().Format("15:04:05.000"), err)
		tsm.Stop()
		return fmt.Errorf("server failed to become healthy: %w", err)
	}
	t.Logf("✅ [%v] Server is healthy!", time.Now().Format("15:04:05.000"))

	// Setup cleanup
	t.Cleanup(func() {
		t.Logf("🧹 [%v] Cleanup: Stopping test server", time.Now().Format("15:04:05.000"))
		tsm.Stop()
	})

	totalTime := time.Since(startTime)
	t.Logf("🎉 [%v] Test server started successfully on port %d (total time: %v)", time.Now().Format("15:04:05.000"), tsm.config.Port, totalTime)
	return nil
}

// Stop stops the test server
func (tsm *TestServerManager) Stop() error {
	if tsm.cancel != nil {
		tsm.cancel()
	}
	if tsm.cmd != nil && tsm.cmd.Process != nil {
		pid := tsm.cmd.Process.Pid

		// In CI environments, use more aggressive cleanup
		if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
			// Force kill immediately in CI to avoid I/O timeout issues
			if killErr := tsm.cmd.Process.Kill(); killErr != nil {
				// Process might already be dead, which is fine
				if !strings.Contains(killErr.Error(), "process already finished") {
					return fmt.Errorf("failed to kill server process %d: %w", pid, killErr)
				}
			}

			// Don't wait for process in CI - just return
			return nil
		}

		// In local environments, try graceful shutdown first
		if err := tsm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Force kill if graceful shutdown fails
			if killErr := tsm.cmd.Process.Kill(); killErr != nil {
				return fmt.Errorf("failed to kill server process %d: %w", pid, killErr)
			}
		}

		// Wait for process to finish with shorter timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // Reduced from 5s
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- tsm.cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			// Force kill on timeout
			tsm.cmd.Process.Kill()
			return nil // Don't return error for timeout in tests
		case err := <-done:
			// Ignore expected exit codes from interrupted processes
			if err != nil && err.Error() != "signal: killed" && err.Error() != "exit status 1" {
				// Don't return error for expected process termination
				return nil
			}
		}
	}

	return nil
}

// IsHealthy checks if the server is healthy
func (tsm *TestServerManager) IsHealthy() bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%d%s", tsm.config.Port, tsm.config.HealthEndpoint)
	resp, err := client.Get(url)
	if err != nil {
		// Don't log every health check failure to avoid spam, but provide info for debugging
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// waitForHealth waits for the server to become healthy
func (tsm *TestServerManager) waitForHealth() error {
	timeout := time.After(tsm.config.StartupTimeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()
	attemptCount := 0
	url := fmt.Sprintf("http://localhost:%d%s", tsm.config.Port, tsm.config.HealthEndpoint)

	for {
		select {
		case <-timeout:
			elapsed := time.Since(startTime)
			// Provide detailed failure information
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(url)
			var statusInfo string
			if err != nil {
				statusInfo = fmt.Sprintf("connection error: %v", err)
			} else {
				statusInfo = fmt.Sprintf("HTTP %d", resp.StatusCode)
				resp.Body.Close()
			}

			// Check if server process is still running
			processStatus := "unknown"
			if tsm.cmd != nil && tsm.cmd.Process != nil {
				if process, err := os.FindProcess(tsm.cmd.Process.Pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						processStatus = "running"
					} else {
						processStatus = "not running"
					}
				}
			}

			return fmt.Errorf("server health check timeout after %v (%d attempts, elapsed: %v, last status: %s, process: %s, url: %s)",
				tsm.config.StartupTimeout, attemptCount, elapsed, statusInfo, processStatus, url)
		case <-ticker.C:
			attemptCount++
			if attemptCount%10 == 0 { // Log every 5 seconds (10 * 500ms)
				elapsed := time.Since(startTime)
				fmt.Printf("⏳ [%v] Health check attempt %d (elapsed: %v, url: %s)\n",
					time.Now().Format("15:04:05.000"), attemptCount, elapsed, url)
			}
			if tsm.IsHealthy() {
				elapsed := time.Since(startTime)
				fmt.Printf("✅ [%v] Server became healthy after %d attempts (elapsed: %v)\n",
					time.Now().Format("15:04:05.000"), attemptCount, elapsed)
				return nil
			}
		}
	}
}

// GetBaseURL returns the base URL for the test server
func (tsm *TestServerManager) GetBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", tsm.config.Port)
}

// MakeRequest makes an HTTP request to the test server
func (tsm *TestServerManager) MakeRequest(method, path string, body interface{}) (*http.Response, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("%s%s", tsm.GetBaseURL(), path)

	var req *http.Request
	var err error

	if body != nil {
		// Handle request body serialization here if needed
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return client.Do(req)
}
