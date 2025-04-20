package testing

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type TestSuite struct {
	Name     string
	Command  string
	Args     []string
	Parallel bool
}

// TestCoordinator manages test execution and reporting
type TestCoordinator struct {
	suites    []TestSuite
	outputDir string
	mu        sync.Mutex
}

// NewTestCoordinator creates a new test coordinator
func NewTestCoordinator(outputDir string) *TestCoordinator {
	return &TestCoordinator{
		outputDir: outputDir,
	}
}

// AddSuite adds a test suite to be executed
func (tc *TestCoordinator) AddSuite(suite TestSuite) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.suites = append(tc.suites, suite)
}

// RunTests executes all test suites
func (tc *TestCoordinator) RunTests() error {
	var wg sync.WaitGroup
	results := make(map[string]error)
	resultsMu := sync.Mutex{}

	for _, suite := range tc.suites {
		if suite.Parallel {
			wg.Add(1)
			go func(s TestSuite) {
				defer wg.Done()
				err := tc.runSuite(s)
				resultsMu.Lock()
				results[s.Name] = err
				resultsMu.Unlock()
			}(suite)
		} else {
			err := tc.runSuite(suite)
			results[suite.Name] = err
		}
	}

	wg.Wait()

	// Process and aggregate results
	return tc.generateReport(results)
}

func (tc *TestCoordinator) runSuite(suite TestSuite) error {
	log.Printf("Running test suite: %s\n", suite.Name)

	cmd := exec.Command(suite.Command, suite.Args...)
	out, err := cmd.CombinedOutput()

	outputFile := filepath.Join(tc.outputDir, fmt.Sprintf("%s.log", suite.Name))
	if err := os.WriteFile(outputFile, out, 0644); err != nil {
		log.Printf("Failed to write output for %s: %v", suite.Name, err)
	}

	return err
}

func (tc *TestCoordinator) generateReport(results map[string]error) error {
	// Generate unified HTML report
	report := NewTestReport(tc.outputDir)
	return report.Generate(results)
}
