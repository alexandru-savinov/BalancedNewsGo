package llm

import (
	"encoding/json"
	"fmt"
	"log"
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

// LoadCompositeScoreConfig loads the composite score configuration from file each time it's called.
// It prioritizes loading relative to the source code structure, which is more reliable during tests.
func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	var configPath string
	var err error

	// 1. Try finding project root relative to source code
	projectRoot, srcErr := getConfigDir()
	if srcErr == nil {
		configPath = filepath.Join(projectRoot, "configs", "composite_score_config.json")
	} else {
		log.Printf("Warning: could not find project root relative to source: %v. Trying other paths.", srcErr)
		// 2. Fallback: Try paths relative to current working directory
		configPaths := []string{
			"configs/composite_score_config.json",
			"../configs/composite_score_config.json",
			"../../configs/composite_score_config.json",
		}
		for _, p := range configPaths {
			if _, e := os.Stat(p); e == nil {
				configPath = p
				break
			}
		}

		// 3. Fallback: Try paths relative to executable (less reliable for tests)
		if configPath == "" {
			execPath, e := os.Executable()
			if e == nil {
				execDir := filepath.Dir(execPath)
				execPaths := []string{
					filepath.Join(execDir, "configs/composite_score_config.json"),
					filepath.Join(execDir, "../configs/composite_score_config.json"),
				}
				for _, p := range execPaths {
					if _, e := os.Stat(p); e == nil {
						configPath = p
						break
					}
				}
			}
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("could not find composite score config file in any standard location")
	}

	log.Printf("Attempting to load composite score config from: %s", configPath)
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("opening composite score config %q: %w", configPath, err)
	}
	defer func() { _ = f.Close() }()

	decoder := json.NewDecoder(f)
	var cfg CompositeScoreConfig
	if e := decoder.Decode(&cfg); e != nil {
		return nil, fmt.Errorf("decoding composite score config %q: %w", configPath, e)
	}
	if len(cfg.Models) == 0 {
		return nil, fmt.Errorf("composite score config %q loaded but Models array is null or empty", configPath)
	}
	log.Printf("Successfully loaded and parsed composite score config from: %s", configPath)
	return &cfg, nil // Return newly loaded config and nil error
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
