package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
)

func main() {
	ctx := context.Background()

	// Initialize embedder (using Ollama - make sure it's running)
	embedderConfig := &config.EmbedderProviderConfig{
		Type:       "ollama",
		Host:       "http://localhost:11434",
		Model:      "nomic-embed-text",
		Timeout:    60, // 60 seconds timeout
		MaxRetries: 3,
	}

	embedder, err := embedders.NewOllamaEmbedderFromConfig(embedderConfig)
	if err != nil {
		fmt.Printf("Failed to create embedder: %v\n", err)
		os.Exit(1)
	}
	defer embedder.Close()

	// Initialize Qdrant database
	dbConfig := &config.VectorStoreConfig{
		Type: "qdrant",
		Host: "localhost",
		Port: 6334,
	}

	db, err := databases.NewQdrantDatabaseProviderFromConfig(dbConfig)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Collection names (matching document store names)
	cookingCollection := "cooking_docs"
	programmingCollection := "programming_docs"

	// Populate cooking documents
	fmt.Println("📚 Indexing cooking documents...")
	if err := indexFolder(ctx, db, embedder, "test-docs/cooking", cookingCollection); err != nil {
		fmt.Printf("Error indexing cooking docs: %v\n", err)
		os.Exit(1)
	}

	// Populate programming documents
	fmt.Println("📚 Indexing programming documents...")
	if err := indexFolder(ctx, db, embedder, "test-docs/programming", programmingCollection); err != nil {
		fmt.Printf("Error indexing programming docs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully populated Qdrant collections!")
	fmt.Printf("   - Collection: %s\n", cookingCollection)
	fmt.Printf("   - Collection: %s\n", programmingCollection)
}

func indexFolder(ctx context.Context, db databases.DatabaseProvider, embedder embedders.EmbedderProvider, folderPath string, collectionName string) error {
	// Get absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Walk through all files in the folder
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\n", path, err)
			return nil // Continue with other files
		}

		// Get relative path for document ID
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		// Create embedding
		text := string(content)
		vector, err := embedder.Embed(text)
		if err != nil {
			return fmt.Errorf("failed to create embedding for %s: %w", path, err)
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"content":       text,
			"path":          relPath,
			"source_path":   relPath,
			"name":          filepath.Base(path),
			"type":          "text",
			"size":          info.Size(),
			"last_modified": info.ModTime().Unix(),
			"store_name":    collectionName,
			"source_type":   "directory",
			"indexed_at":    time.Now().Unix(),
		}

		// Create document ID using MD5 hash (like document store does)
		docKey := fmt.Sprintf("%s:%s", collectionName, relPath)
		hash := md5.Sum([]byte(docKey))
		docID := uuid.NewMD5(uuid.Nil, hash[:]).String()

		// Upsert to Qdrant
		if err := db.Upsert(ctx, collectionName, docID, vector, metadata); err != nil {
			return fmt.Errorf("failed to upsert document %s: %w", docID, err)
		}

		fmt.Printf("  ✓ Indexed: %s\n", relPath)
		return nil
	})

	return err
}
