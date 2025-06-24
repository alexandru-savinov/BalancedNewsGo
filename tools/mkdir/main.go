// tools/mkdir.go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run tools/mkdir.go <dirname>")
		os.Exit(1)
	}
	dirToCreate := os.Args[1]
	err := os.MkdirAll(dirToCreate, 0750) // Use secure directory permissions
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dirToCreate, err)
		os.Exit(1)
	}
	fmt.Printf("Ensured directory exists: %s\n", dirToCreate)
}
