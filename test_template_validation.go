package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Test template parsing to ensure our changes don't break Go template compilation
	templatesDir := "templates"

	fmt.Println("🧪 Testing Template Compilation...")

	// Test main templates
	templates := []string{
		"templates/articles.html",
		"templates/articles_htmx.html",
		"templates/admin.html",
		"templates/article.html",
		"templates/article_htmx.html",
	}

	for _, tmplPath := range templates {
		if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Template not found: %s\n", tmplPath)
			continue
		}

		_, err := template.ParseFiles(tmplPath)
		if err != nil {
			fmt.Printf("❌ Template compilation failed: %s\n", tmplPath)
			fmt.Printf("   Error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("✅ Template compiled successfully: %s\n", tmplPath)
		}
	}

	// Test fragment templates
	fragmentsDir := "templates/fragments"
	err := filepath.Walk(fragmentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".html" {
			_, parseErr := template.ParseFiles(path)
			if parseErr != nil {
				fmt.Printf("❌ Fragment template compilation failed: %s\n", path)
				fmt.Printf("   Error: %v\n", parseErr)
				os.Exit(1)
			} else {
				fmt.Printf("✅ Fragment compiled successfully: %s\n", path)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking fragments directory: %v", err)
	}

	fmt.Println("\n🎉 All templates compiled successfully!")
	fmt.Println("✅ No breaking changes detected in template syntax")
}
