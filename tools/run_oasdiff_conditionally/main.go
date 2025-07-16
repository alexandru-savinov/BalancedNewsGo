package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// getGoEnvVar executes "go env <var_name>" and returns its output.
func getGoEnvVar(varName string) (string, error) {
	// Validate varName to prevent command injection - only allow known safe values
	allowedVars := map[string]bool{
		"GOBIN":  true,
		"GOPATH": true,
		"GOROOT": true,
		"GOOS":   true,
		"GOARCH": true,
	}
	if !allowedVars[varName] {
		return "", fmt.Errorf("invalid go env variable name: %s", varName)
	}

	// #nosec G204 - varName is validated against allowlist, go command is standard development tool
	cmd := exec.Command("go", "env", varName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get go env %s: %w", varName, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// findOasdiff attempts to locate the oasdiff executable.
func findOasdiff() (string, error) {
	exeName := "oasdiff"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}

	// 1. Check PATH
	oasdiffPath, err := exec.LookPath(exeName)
	if err == nil {
		return oasdiffPath, nil
	}

	checkedPaths := []string{"system PATH"}

	// 2. Check GOBIN
	gobin, gobinErr := getGoEnvVar("GOBIN")
	if gobinErr == nil && gobin != "" {
		tryPath := filepath.Join(gobin, exeName)
		checkedPaths = append(checkedPaths, tryPath)
		if _, err := os.Stat(tryPath); err == nil {
			return tryPath, nil
		}
	} else if gobinErr != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Error getting GOBIN: %v\n", gobinErr)
	}

	// 3. Check GOPATH/bin
	gopath, gopathErr := getGoEnvVar("GOPATH")
	if gopathErr == nil && gopath != "" {
		tryPath := filepath.Join(gopath, "bin", exeName)
		checkedPaths = append(checkedPaths, tryPath)
		if _, err := os.Stat(tryPath); err == nil {
			return tryPath, nil
		}
	} else if gopathErr != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Error getting GOPATH: %v\n", gopathErr)
	}

	return "", fmt.Errorf("'%s' command not found. Checked locations: %s. "+
		"Please ensure oasdiff is installed (go install github.com/oasdiff/oasdiff@latest) "+
		"and your PATH, GOBIN, or GOPATH/bin is configured correctly. "+
		"Refer to README.md for details", exeName, strings.Join(checkedPaths, ", "))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: go run run_oasdiff_conditionally.go <baseline_spec> <current_spec>")
		os.Exit(2)
	}
	baselineSpecPath := os.Args[1]
	currentSpecPath := os.Args[2]

	_, err := os.Stat(baselineSpecPath)
	if os.IsNotExist(err) {
		fmt.Printf("INFO: No previous API specification (%s) found. Skipping breaking change detection.\n", baselineSpecPath)
		os.Exit(0) // Success, as skipping is a valid outcome
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking baseline spec %s: %v\n", baselineSpecPath, err)
		os.Exit(1) // Error during file check
	}

	// Baseline exists, proceed to check for oasdiff and run it
	fmt.Printf("INFO: Previous API specification (%s) found. Comparing with %s...\n", baselineSpecPath, currentSpecPath)

	oasdiffExecutable, err := findOasdiff()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: "+err.Error())
		os.Exit(1)
	}
	// fmt.Fprintf(os.Stderr, "DEBUG: Using oasdiff executable at: %s\n", oasdiffExecutable) // Optional debug line

	cmd := exec.Command(oasdiffExecutable, "breaking", baselineSpecPath, currentSpecPath) // #nosec G204 - controlled input from command line
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// oasdiff returns exit code 1 for breaking changes found, 2 for other errors.
			// We consider finding breaking changes as a "successful" run of the tool in terms of process,
			// but the build/CI should interpret this exit code.
			fmt.Fprintf(os.Stderr, "INFO: oasdiff completed. Exit code: %d "+
				"(1 means breaking changes found, >1 means oasdiff error)\n", exitErr.ExitCode())
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error running oasdiff: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("INFO: oasdiff completed successfully (no breaking changes or oasdiff internal errors " +
		"reported to stdout/stderr by the tool itself).")
	os.Exit(0)
}
