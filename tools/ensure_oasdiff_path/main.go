package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.LookPath("oasdiff")
	if err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: 'oasdiff' command not found in PATH. "+
			"Please ensure it is installed and your PATH is configured correctly.")
		fmt.Fprintln(os.Stderr, "         You can install it with: go install github.com/Tufin/oasdiff/cmd/oasdiff@latest")
		fmt.Fprintln(os.Stderr, "         Refer to README.md for more details.")
		// Exit 0 anyway so the makefile can attempt to run oasdiff and fail there if truly not found,
		// allowing oasdiff's own error messages.
		os.Exit(0)
	}
	os.Exit(0) // Found in PATH
}
