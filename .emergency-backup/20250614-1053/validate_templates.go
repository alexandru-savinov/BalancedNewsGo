package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

func main() {
	templateDir := "templates"

	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".html" {
			fmt.Printf("Validating template: %s\n", path)

			// Try to parse the template
			_, err := template.ParseFiles(path)
			if err != nil {
				fmt.Printf("❌ ERROR in %s: %v\n", path, err)
				return nil // Continue checking other templates
			}

			fmt.Printf("✅ OK: %s\n", path)
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n✅ Template validation complete")
}
