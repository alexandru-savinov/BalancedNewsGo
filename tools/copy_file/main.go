package main

import (
	"fmt"
	"io"
	"log"
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
	defer func() {
		if err := sourceFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing source file: %v\n", err)
		}
	}()

	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination file %s: %v\n", destPath, err)
		os.Exit(1)
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing destination file: %v\n", err)
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying from %s to %s: %v\n", sourcePath, destPath, err)
		log.Printf("Error: %v\n", err)
		// sourceFile and destFile are closed by defer, but if we exit here, defers won't run.
		// Explicitly close them if they were opened.
		if sourceFile != nil {
			if err := sourceFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing source file before exit: %v\n", err)
			}
		}
		if destFile != nil {
			if err := destFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing destination file before exit: %v\n", err)
			}
		}
		os.Exit(1)
	}
	// fmt.Printf("Copied %s to %s\n", sourcePath, destPath) // Optional: success message
	os.Exit(0)
}
