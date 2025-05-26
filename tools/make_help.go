package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
)

func main() {
	file, err := os.Open("Makefile")
	if err != nil {
		fmt.Println("Could not open Makefile:", err)
		os.Exit(1)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	re := regexp.MustCompile(`^([a-zA-Z0-9_-]+):.*?## (.*)`)
	var targets [][2]string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if m := re.FindStringSubmatch(line); m != nil {
			targets = append(targets, [2]string{m[1], m[2]})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i][0] < targets[j][0] })

	fmt.Println("\nTargets:")
	for _, t := range targets {
		fmt.Printf("  %-15s %s\n", t[0], t[1])
	}
}
