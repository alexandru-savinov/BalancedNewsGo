package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var (
	// fileCompositeScoreConfig caches the config loaded from file
	fileCompositeScoreConfig     *CompositeScoreConfig
	fileCompositeScoreConfigOnce sync.Once
)

// LoadCompositeScoreConfig loads the composite score configuration from file
func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	var err error
	fileCompositeScoreConfigOnce.Do(func() {
		// Try multiple possible locations for the config file
		configPaths := []string{
			"configs/composite_score_config.json",                     // Relative to current directory
			"../configs/composite_score_config.json",                  // One level up
			"../../configs/composite_score_config.json",               // Two levels up
			filepath.Join(".", "configs/composite_score_config.json"), // Explicit relative path
		}

		// Get the executable's directory
		execPath, e := os.Executable()
		if e == nil {
			execDir := filepath.Dir(execPath)
			// Add paths relative to the executable
			configPaths = append(configPaths,
				filepath.Join(execDir, "configs/composite_score_config.json"),
				filepath.Join(execDir, "../configs/composite_score_config.json"),
			)
		}

		var f *os.File
		var openedPath string

		// Try each path until we find one that works
		for _, path := range configPaths {
			if f2, e := os.Open(path); e == nil {
				f = f2
				openedPath = path
				break
			}
		}

		if f == nil {
			err = fmt.Errorf("could not find composite score config in any of the expected locations")
			return
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		var cfg CompositeScoreConfig
		if e := decoder.Decode(&cfg); e != nil {
			err = fmt.Errorf("decoding composite score config %q: %w", openedPath, e)
			return
		}
		if len(cfg.Models) == 0 {
			err = fmt.Errorf("composite score config %q loaded but contains no models", openedPath)
			return
		}
		fileCompositeScoreConfig = &cfg
	})
	if err != nil {
		return nil, err
	}
	return fileCompositeScoreConfig, nil
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
