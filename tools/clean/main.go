package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Directories to remove
	dirsToRemove := []string{"./bin", "./coverage"}

	// Remove directories
	for _, dir := range dirsToRemove {
		fmt.Printf("Removing directory: %s\n", dir)
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Printf("Warning: Could not remove %s: %v\n", dir, err)
		}
	}

	// Find and remove coverage files
	matches, err := filepath.Glob("coverage*.out")
	if err != nil {
		fmt.Printf("Warning: Error finding coverage files: %v\n", err)
	} else {
		for _, file := range matches {
			fmt.Printf("Removing file: %s\n", file)
			err := os.Remove(file)
			if err != nil {
				fmt.Printf("Warning: Could not remove %s: %v\n", file, err)
			}
		}
	}

	fmt.Println("Clean operation complete")
}
