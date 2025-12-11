// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel

// Package server provides the HTTP and gRPC server implementation for Hector.
//
// The server embeds a web UI that is built from the ui/ directory.
// The UI build is automated using go generate.
//
// To build the complete server with UI:
//
//	go generate ./pkg/server
//	go build ./cmd/hector
//
// Or use the Makefile targets which handle this automatically:
//
//	make build        # Development build
//	make build-release # Production build
//	make install      # Install to GOPATH/bin
package server

//go:generate go run generate.go
