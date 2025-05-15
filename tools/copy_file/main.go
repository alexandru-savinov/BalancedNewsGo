package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: go run copy_file.go <source> <destination>")
		os.Exit(2)
	}
	sourcePath := os.Args[1]
	destPath := os.Args[2]

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening source file %s: %v\n", sourcePath, err)
		os.Exit(1)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination file %s: %v\n", destPath, err)
		os.Exit(1)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying from %s to %s: %v\n", sourcePath, destPath, err)
		os.Exit(1)
	}
	// fmt.Printf("Copied %s to %s\n", sourcePath, destPath) // Optional: success message
	os.Exit(0)
}
