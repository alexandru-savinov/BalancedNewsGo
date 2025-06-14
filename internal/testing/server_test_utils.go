package testing

import (
	"context"
	"fmt"
	"net/http"
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
	return TestServerConfig{
		Port:            8080,
		StartupTimeout:  30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		HealthEndpoint:  "/healthz",
		ServerCommand:   []string{"go", "run", "./cmd/server"},
		Environment:     make(map[string]string),
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

	// Set environment variables
	for key, value := range tsm.config.Environment {
		tsm.cmd.Env = append(tsm.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

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
		// Try graceful shutdown first
		if err := tsm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// Force kill if graceful shutdown fails
			if killErr := tsm.cmd.Process.Kill(); killErr != nil {
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
			tsm.cmd.Process.Kill()
			return fmt.Errorf("server shutdown timeout")
		case err := <-done:
			if err != nil && err.Error() != "signal: killed" {
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
