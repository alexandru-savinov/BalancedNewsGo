package testing

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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

	// Use compiled binary in test environments to avoid file locking issues
	serverCommand := []string{"go", "run", "./cmd/server"}
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
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

	// Check if server is already running
	if tsm.IsHealthy() {
		t.Log("Server already running and healthy")
		return nil
	}

	// Create context for server process
	ctx, cancel := context.WithCancel(context.Background())
	tsm.cancel = cancel

	// Setup server command
	tsm.cmd = exec.CommandContext(ctx, tsm.config.ServerCommand[0], tsm.config.ServerCommand[1:]...)
	tsm.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // isolate process group

	// Set working directory to project root
	// Try to detect if we're in tests directory and go up, otherwise use current directory
	workingDir := "."
	if _, err := os.Stat("../cmd/server"); err == nil {
		workingDir = ".."
	}
	tsm.cmd.Dir = workingDir

	// Copy current environment and add test-specific variables
	tsm.cmd.Env = os.Environ()
	for key, value := range tsm.config.Environment {
		tsm.cmd.Env = append(tsm.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture server output for debugging
	tsm.cmd.Stdout = os.Stdout
	tsm.cmd.Stderr = os.Stderr

	// Start the server
	if err := tsm.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait for server to be healthy
	if err := tsm.waitForHealth(); err != nil {
		tsm.Stop()
		return fmt.Errorf("server failed to become healthy: %w", err)
	}

	// Setup cleanup
	t.Cleanup(func() {
		tsm.Stop()
	})

	t.Logf("Test server started successfully on port %d", tsm.config.Port)
	return nil
}

// Stop stops the test server
func (tsm *TestServerManager) Stop() error {
	if tsm.cancel != nil {
		tsm.cancel()
	}
	if tsm.cmd != nil && tsm.cmd.Process != nil {
		// Try graceful shutdown of the entire process group first
		pgid, _ := syscall.Getpgid(tsm.cmd.Process.Pid)
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			// Force kill if graceful shutdown fails
			if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
				return fmt.Errorf("failed to kill server process: %w", killErr)
			}
		}

		// Wait for process to finish
		ctx, cancel := context.WithTimeout(context.Background(), tsm.config.ShutdownTimeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- tsm.cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			pgid, _ := syscall.Getpgid(tsm.cmd.Process.Pid)
			syscall.Kill(-pgid, syscall.SIGKILL)
			return fmt.Errorf("server shutdown timeout")
		case err := <-done:
			// Ignore expected exit codes from interrupted processes
			if err != nil && err.Error() != "signal: killed" && err.Error() != "exit status 1" {
				return fmt.Errorf("server process error: %w", err)
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

	for {
		select {
		case <-timeout:
			return fmt.Errorf("server health check timeout after %v", tsm.config.StartupTimeout)
		case <-ticker.C:
			if tsm.IsHealthy() {
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
