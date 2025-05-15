package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run check_file_exists.go <filepath>")
		os.Exit(2) // Indicate incorrect usage
	}
	filepath := os.Args[1]
	if _, err := os.Stat(filepath); err == nil {
		// File exists
		os.Exit(0)
	} else if os.IsNotExist(err) {
		// File does not exist
		os.Exit(1)
	} else {
		// Other error (e.g., permission issues)
		fmt.Fprintf(os.Stderr, "Error checking file %s: %v\n", filepath, err)
		os.Exit(3) // Indicate other error
	}
}
