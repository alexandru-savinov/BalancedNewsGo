package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Label constants for perspectives
const (
	LabelLeft    = "left"
	LabelCenter  = "center"
	LabelRight   = "right"
	LabelNeutral = "neutral"
)

// CompositeScoreConfig defines the structure for composite score calculation configuration
type CompositeScoreConfig struct {
	Models            []ModelConfig      `json:"models"`
	Formula           string             `json:"formula"` // "average" or "weighted"
	ConfidenceMethod  string             `json:"confidence_method"`
	MinScore          float64            `json:"min_score"`
	MaxScore          float64            `json:"max_score"`
	DefaultMissing    float64            `json:"default_missing"`
	MinConfidence     float64            `json:"min_confidence"`
	MaxConfidence     float64            `json:"max_confidence"`
	HandleInvalid     string             `json:"handle_invalid"` // "default" or "ignore"
	Weights           map[string]float64 `json:"weights"`        // Optional: Perspective weights for "weighted" formula
	ArticleIDForDebug int64              `json:"-"`              // Temporary field for debugging logs, ignored by JSON
}

// ModelConfig defines configuration for a single model within the composite score
type ModelConfig struct {
	ModelName   string  `json:"modelName"`
	Perspective string  `json:"perspective"`
	Weight      float64 `json:"weight"`
	URL         string  `json:"url"`
}

// LoadCompositeScoreConfig loads the configuration from a JSON file
func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	// Try multiple possible locations for the config file
	var configPath string
	var err error

	// First try: relative to current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
	} else {
		configPath = filepath.Join(wd, "configs", "composite_score_config.json")
		if _, err := os.Stat(configPath); err == nil {
			log.Printf("Found composite score config at: %s", configPath)
			return loadConfigFromPath(configPath)
		}
	}

	// Second try: absolute path (for Docker containers)
	configPath = "/configs/composite_score_config.json"
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("Found composite score config at: %s", configPath)
		return loadConfigFromPath(configPath)
	}

	// Third try: relative to executable
	configPath = "configs/composite_score_config.json"
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("Found composite score config at: %s", configPath)
		return loadConfigFromPath(configPath)
	}

	log.Printf("Could not find composite score config file in any of the expected locations")
	return nil, fmt.Errorf("composite score config file not found")
}

func loadConfigFromPath(configPath string) (*CompositeScoreConfig, error) {
	log.Printf("Attempting to load composite score config from: %s", configPath)

	bytes, err := os.ReadFile(configPath) // #nosec G304 - configPath is from application configuration, controlled input
	if err != nil {
		log.Printf("Error reading config file %s: %v", configPath, err)
		return nil, err
	}

	var config CompositeScoreConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		log.Printf("Error parsing config file %s: %v", configPath, err)
		return nil, err
	}

	log.Printf("Successfully loaded and parsed composite score config from: %s", configPath)
	return &config, nil
}
