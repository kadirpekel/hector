// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package provider defines the config source abstraction.
//
// Providers load configuration from various sources (file, consul, etcd, etc.)
// and support watching for changes.
package provider

import (
	"context"
	"fmt"
)

// Type identifies the config source type.
type Type string

const (
	TypeFile      Type = "file"
	TypeConsul    Type = "consul"
	TypeEtcd      Type = "etcd"
	TypeZookeeper Type = "zookeeper"
)

// ParseType converts a string to a Type.
func ParseType(s string) (Type, error) {
	switch s {
	case "file", "":
		return TypeFile, nil
	case "consul":
		return TypeConsul, nil
	case "etcd":
		return TypeEtcd, nil
	case "zookeeper", "zk":
		return TypeZookeeper, nil
	default:
		return "", fmt.Errorf("unknown provider type: %s", s)
	}
}

// Provider abstracts config sources.
//
// Implementations must be safe for concurrent use.
type Provider interface {
	// Type returns the provider type for logging/debugging.
	Type() Type

	// Load reads raw config bytes from the source.
	Load(ctx context.Context) ([]byte, error)

	// Watch starts watching for changes and signals via the returned channel.
	// The channel receives a value when config changes.
	// Cancel the context to stop watching.
	// Returns nil channel if watching is not supported.
	Watch(ctx context.Context) (<-chan struct{}, error)

	// Close releases any resources held by the provider.
	Close() error
}

// ProviderConfig configures provider creation.
type ProviderConfig struct {
	// Type specifies the provider type (file, consul, etcd, zookeeper).
	Type Type

	// Path is the config path (file path or key path).
	Path string

	// Endpoints for remote providers (consul, etcd, zookeeper).
	Endpoints []string
}

// New creates a Provider based on ProviderConfig.
func New(opts ProviderConfig) (Provider, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	switch opts.Type {
	case TypeFile, "":
		return NewFileProvider(opts.Path)
	case TypeConsul:
		return nil, fmt.Errorf("consul provider not yet implemented")
	case TypeEtcd:
		return nil, fmt.Errorf("etcd provider not yet implemented")
	case TypeZookeeper:
		return nil, fmt.Errorf("zookeeper provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider type: %s", opts.Type)
	}
}
