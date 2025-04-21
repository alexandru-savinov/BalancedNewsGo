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

// TestReport represents the structure for generating test reports
type TestReport struct {
	outputDir string
}

// NewTestReport creates a new TestReport instance
func NewTestReport(outputDir string) *TestReport {
	return &TestReport{outputDir: outputDir}
}

// Generate creates the final report based on test results
func (tr *TestReport) Generate(results map[string]error) error {
	// Placeholder for report generation logic
	log.Println("Generating test report...")
	for name, err := range results {
		status := "PASS"
		if err != nil {
			status = fmt.Sprintf("FAIL: %v", err)
		}
		log.Printf("Suite: %s, Status: %s\n", name, status)
	}
	// In a real implementation, this would write to an HTML or other format file
	// in tr.outputDir
	log.Println("Report generation complete.")
	return nil // Placeholder return
}
