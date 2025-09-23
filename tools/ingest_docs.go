package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ingest_docs.go <config-file> <documents-directory>")
		fmt.Println("Example: go run ingest_docs.go min.yaml test-documents/")
		os.Exit(1)
	}

	configFile := os.Args[1]
	docsDir := os.Args[2]

	// Load agent configuration
	agent, err := hector.LoadAgentFromFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load agent: %v", err)
	}

	fmt.Printf("Ingesting documents from %s using config %s...\n", docsDir, configFile)

	// Walk through documents directory
	err = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-text files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".txt") {
			return nil
		}

		fmt.Printf("Processing: %s\n", path)

		// Read file content
		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			return nil
		}

		// Use the agent's private method through a helper
		err = ingestDocument(agent, path, string(content))
		if err != nil {
			fmt.Printf("Error ingesting %s: %v\n", path, err)
			return nil
		}

		fmt.Printf("Successfully ingested: %s\n", path)
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	fmt.Println("Document ingestion completed!")
}

// Helper function to ingest a single document
func ingestDocument(agent *hector.Agent, docPath, content string) error {
	ctx := context.Background()

	// Create metadata with document information
	metadata := map[string]interface{}{
		"filename": filepath.Base(docPath),
		"path":     docPath,
		"size":     len(content),
	}

	// Generate a UUID based on the file path for consistent IDs
	hash := md5.Sum([]byte(docPath))
	docID := uuid.NewMD5(uuid.Nil, hash[:]).String()

	// Get the search engine from agent and ingest the document
	searchEngine := agent.GetSearchEngine()
	if searchEngine == nil {
		return fmt.Errorf("search engine not configured")
	}
	return searchEngine.IngestDocument(ctx, docID, content, metadata)
}
