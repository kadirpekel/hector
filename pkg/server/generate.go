// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel

//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// This file is used by `go generate` to build the UI before embedding.
// It ensures the static/index.html file exists before the Go build process.

func main() {
	// Get the project root (two directories up from pkg/server)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}

	uiDir := filepath.Join(projectRoot, "ui")
	staticDir := filepath.Join(projectRoot, "pkg", "server", "static")
	indexPath := filepath.Join(staticDir, "index.html")

	fmt.Println("Building UI assets...")
	fmt.Printf("  UI directory: %s\n", uiDir)
	fmt.Printf("  Target: %s\n", indexPath)

	// Check if UI directory exists
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		log.Fatalf("UI directory not found: %s", uiDir)
	}

	// Install npm dependencies
	fmt.Println("  Installing npm dependencies...")
	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = uiDir
	npmInstall.Stdout = os.Stdout
	npmInstall.Stderr = os.Stderr
	if err := npmInstall.Run(); err != nil {
		log.Fatalf("Failed to run npm install: %v", err)
	}

	// Build the UI
	fmt.Println("  Building UI with Vite...")
	npmBuild := exec.Command("npm", "run", "build")
	npmBuild.Dir = uiDir
	npmBuild.Stdout = os.Stdout
	npmBuild.Stderr = os.Stderr
	if err := npmBuild.Run(); err != nil {
		log.Fatalf("Failed to run npm build: %v", err)
	}

	// Create static directory if it doesn't exist
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		log.Fatalf("Failed to create static directory: %v", err)
	}

	// Copy the built file
	fmt.Println("  Copying index.html to static assets...")
	sourcePath := filepath.Join(uiDir, "dist", "index.html")
	if err := copyFile(sourcePath, indexPath); err != nil {
		log.Fatalf("Failed to copy index.html: %v", err)
	}

	// Verify the file was created
	if stat, err := os.Stat(indexPath); err != nil {
		log.Fatalf("Failed to verify index.html: %v", err)
	} else {
		fmt.Printf("✅ UI built successfully (%d MB)\n", stat.Size()/(1024*1024))
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
