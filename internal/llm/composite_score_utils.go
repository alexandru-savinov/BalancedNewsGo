package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// getConfigDir attempts to find the directory containing the configs relative to the source file.
func getConfigDir() (string, error) {
	_, filename, _, ok := runtime.Caller(1) // Get caller's file path
	if !ok {
		return "", fmt.Errorf("could not get caller information")
	}

	// Assume config is in `configs` dir relative to project root.
	// We need to walk up from the source file's directory.
	dir := filepath.Dir(filename)
	for i := 0; i < 10; i++ { // Limit walk depth
		configPath := filepath.Join(dir, "configs", "composite_score_config.json")
		if _, err := os.Stat(configPath); err == nil {
			// Found the config file, so return the parent dir (project root)
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // Reached root directory
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find project root containing configs directory relative to source file %s", filename)
}

// minNonNil returns the minimum value from a map of float64s
func minNonNil(m map[string]float64, def float64) float64 {
	min := def
	first := true
	for _, v := range m {
		if first || v < min {
			min = v
			first = false
		}
	}
	return min
}

// maxNonNil returns the maximum value from a map of float64s
func maxNonNil(m map[string]float64, def float64) float64 {
	max := def
	first := true
	for _, v := range m {
		if first || v > max {
			max = v
			first = false
		}
	}
	return max
}

// scoreSpread calculates the difference between the maximum and minimum score
func scoreSpread(m map[string]float64) float64 {
	vals := []float64{}
	for _, v := range m {
		vals = append(vals, v)
	}
	if len(vals) < 2 {
		return 0.0
	}
	sort.Float64s(vals)
	return vals[len(vals)-1] - vals[0]
}
