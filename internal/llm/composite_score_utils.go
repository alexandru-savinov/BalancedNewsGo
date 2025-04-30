package llm

import (
	"encoding/json"
	"fmt"
	"os"
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
		const configPath = "configs/composite_score_config.json"
		f, e := os.Open(configPath)
		if e != nil {
			err = fmt.Errorf("opening composite score config %q: %w", configPath, e)
			return
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		var cfg CompositeScoreConfig
		if e := decoder.Decode(&cfg); e != nil {
			err = fmt.Errorf("decoding composite score config %q: %w", configPath, e)
			return
		}
		if len(cfg.Models) == 0 {
			err = fmt.Errorf("composite score config %q loaded but contains no models", configPath)
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
