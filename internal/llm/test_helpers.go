//go:build test

package llm

import (
	"os"
	"path/filepath"
)

func testEnsureProjectRoot() {
	dir, _ := os.Getwd()
	for !fileExists(filepath.Join(dir, "configs", "composite_score_config.json")) && dir != filepath.Dir(dir) {
		dir = filepath.Dir(dir)
	}
	_ = os.Chdir(dir)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
