# Hector Makefile
# Build and release management for the Hector AI agent platform

.PHONY: help build install test clean fmt vet lint release version test-coverage test-coverage-summary test-package test-race test-verbose dev ci install-lint quality pre-commit

# Default target
help:
	@echo "Hector Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build the hector binary"
	@echo "  install   - Install hector to GOPATH/bin"
	@echo "  test      - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-coverage-summary - Run tests with coverage summary"
	@echo "  test-package - Run tests for specific package (PACKAGE=pkg/name)"
	@echo "  test-race - Run tests with race detection"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  clean     - Clean build artifacts"
	@echo "  fmt       - Format Go code"
	@echo "  vet       - Run go vet"
	@echo "  lint      - Run golangci-lint (if installed)"
	@echo "  release   - Build release binaries"
	@echo "  version   - Show version information"
	@echo "  deps      - Download dependencies"
	@echo "  mod-tidy  - Tidy go.mod"

# Build the binary
build:
	@echo "Building hector..."
	go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o hector ./cmd/hector

# Install to GOPATH/bin
install:
	@echo "Installing hector..."
	go install -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" ./cmd/hector

# Install to system PATH (requires sudo)
install-system:
	@echo "Installing hector to /usr/local/bin..."
	@sudo cp hector /usr/local/bin/hector
	@echo "Hector installed successfully!"

# Uninstall from system PATH
uninstall:
	@echo "Uninstalling hector from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/hector
	@echo "Hector uninstalled successfully!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage and show summary
test-coverage-summary:
	@echo "Running tests with coverage summary..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f hector
	rm -f coverage.out coverage.html
	go clean

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run linter (if available)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Build release binaries for multiple platforms
release:
	@echo "Building release binaries..."
	@mkdir -p dist
	
	# Linux
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o dist/hector-linux-amd64 ./cmd/hector
	GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o dist/hector-linux-arm64 ./cmd/hector
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o dist/hector-darwin-amd64 ./cmd/hector
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o dist/hector-darwin-arm64 ./cmd/hector
	
	# Windows
	GOOS=windows GOARCH=amd64 go build -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o dist/hector-windows-amd64.exe ./cmd/hector
	
	@echo "Release binaries built in dist/"

# Show version information
version:
	@echo "Version Information:"
	@go run -ldflags "-X 'github.com/kadirpekel/hector.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'github.com/kadirpekel/hector.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" ./cmd/hector version

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Tidy go.mod
mod-tidy:
	@echo "Tidying go.mod..."
	go mod tidy

# Development workflow
dev: fmt vet lint test build
	@echo "Development build complete"

# CI workflow
ci: deps fmt vet lint test
	@echo "CI checks complete"

# Test package coverage
test-package:
	@echo "Running tests for specific package..."
	@if [ -z "$(PACKAGE)" ]; then \
		echo "Usage: make test-package PACKAGE=pkg/config"; \
		exit 1; \
	fi
	go test -v -coverprofile=coverage.out ./$(PACKAGE)/...
	go tool cover -func=coverage.out

# Test with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Test with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		$(MAKE) install-lint; \
	fi
	export PATH=$$PATH:$(shell go env GOPATH)/bin && golangci-lint run --timeout=5m

# Run all quality checks
quality: fmt vet lint test
	@echo "All quality checks passed"

# Pre-commit checks (what CI runs)
pre-commit: deps fmt vet lint test build
	@echo "Pre-commit checks complete - ready to push"
