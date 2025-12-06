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

package rag

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents the health state of a component.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is functioning normally.
	HealthStatusHealthy HealthStatus = "healthy"

	// HealthStatusDegraded indicates the component is functioning but with issues.
	HealthStatusDegraded HealthStatus = "degraded"

	// HealthStatusUnhealthy indicates the component is not functioning.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents the result of a health check.
type HealthCheck struct {
	// Component name.
	Component string `json:"component"`

	// Status of the component.
	Status HealthStatus `json:"status"`

	// Message provides details about the status.
	Message string `json:"message,omitempty"`

	// Latency of the health check.
	Latency time.Duration `json:"latency_ms"`

	// Timestamp of the check.
	Timestamp time.Time `json:"timestamp"`

	// Details contains component-specific health information.
	Details map[string]any `json:"details,omitempty"`
}

// IsHealthy returns true if the status is healthy.
func (h HealthCheck) IsHealthy() bool {
	return h.Status == HealthStatusHealthy
}

// DocumentStoreHealth checks the health of a DocumentStore.
func (s *DocumentStore) HealthCheck(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Component: fmt.Sprintf("document_store:%s", s.name),
		Timestamp: start,
		Details:   make(map[string]any),
	}

	// Check source connectivity
	sourceCheck := s.checkSourceHealth(ctx)
	check.Details["source"] = sourceCheck

	// Check engine health
	engineCheck := s.engine.HealthCheck(ctx)
	check.Details["engine"] = engineCheck

	// Determine overall status
	if sourceCheck.Status == HealthStatusUnhealthy || engineCheck.Status == HealthStatusUnhealthy {
		check.Status = HealthStatusUnhealthy
		check.Message = "one or more components unhealthy"
	} else if sourceCheck.Status == HealthStatusDegraded || engineCheck.Status == HealthStatusDegraded {
		check.Status = HealthStatusDegraded
		check.Message = "one or more components degraded"
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "all components healthy"
	}

	// Add metrics
	stats := s.Stats()
	check.Details["indexed_count"] = stats.IndexedCount
	check.Details["watch_enabled"] = stats.WatchEnabled
	check.Details["source_type"] = stats.SourceType

	check.Latency = time.Since(start)
	return check
}

// checkSourceHealth checks if the data source is accessible.
func (s *DocumentStore) checkSourceHealth(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Component: fmt.Sprintf("source:%s", s.source.Type()),
		Timestamp: start,
		Details:   make(map[string]any),
	}

	// For directory sources, check if path is accessible
	if s.source.Type() == "directory" {
		if dirSource, ok := s.source.(*DirectorySource); ok {
			basePath := dirSource.GetBasePath()
			check.Details["path"] = basePath

			// Check if path exists (quick test)
			testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			// Try to get last modified time of base path
			_, err := s.source.GetLastModified(testCtx, basePath)
			if err != nil {
				check.Status = HealthStatusUnhealthy
				check.Message = fmt.Sprintf("cannot access path: %v", err)
			} else {
				check.Status = HealthStatusHealthy
				check.Message = "directory accessible"
			}
		}
	} else {
		// For other sources, assume healthy (would need source-specific checks)
		check.Status = HealthStatusHealthy
		check.Message = "source type healthy"
	}

	check.Latency = time.Since(start)
	return check
}

// SearchEngineHealth checks the health of a SearchEngine.
func (e *SearchEngine) HealthCheck(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Component: fmt.Sprintf("search_engine:%s", e.collection),
		Timestamp: start,
		Details:   make(map[string]any),
	}

	// Check vector provider
	providerCheck := e.checkProviderHealth(ctx)
	check.Details["provider"] = providerCheck

	// Check embedder
	embedderCheck := e.checkEmbedderHealth(ctx)
	check.Details["embedder"] = embedderCheck

	// Determine overall status
	if providerCheck.Status == HealthStatusUnhealthy || embedderCheck.Status == HealthStatusUnhealthy {
		check.Status = HealthStatusUnhealthy
		check.Message = "one or more components unhealthy"
	} else if providerCheck.Status == HealthStatusDegraded || embedderCheck.Status == HealthStatusDegraded {
		check.Status = HealthStatusDegraded
		check.Message = "one or more components degraded"
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "all components healthy"
	}

	// Add component info
	check.Details["provider_name"] = e.provider.Name()
	check.Details["collection"] = e.collection
	check.Details["hyde_enabled"] = e.hyde != nil
	check.Details["reranker_enabled"] = e.reranker != nil
	check.Details["multiquery_enabled"] = e.multiQuery != nil

	check.Latency = time.Since(start)
	return check
}

// checkProviderHealth verifies vector provider connectivity.
func (e *SearchEngine) checkProviderHealth(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Component: fmt.Sprintf("vector_provider:%s", e.provider.Name()),
		Timestamp: start,
		Details:   make(map[string]any),
	}

	// Try a simple search with empty vector to test connectivity
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create a minimal test vector
	testVector := make([]float32, 1536) // Common embedding dimension
	_, err := e.provider.Search(testCtx, e.collection, testVector, 1)

	if err != nil {
		// Some errors are expected (e.g., collection doesn't exist yet)
		errStr := err.Error()
		if containsAny(errStr, []string{"not found", "does not exist", "empty"}) {
			check.Status = HealthStatusHealthy
			check.Message = "provider reachable (collection may be empty)"
		} else {
			check.Status = HealthStatusUnhealthy
			check.Message = fmt.Sprintf("provider error: %v", err)
		}
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "provider healthy"
	}

	check.Latency = time.Since(start)
	return check
}

// checkEmbedderHealth verifies embedder connectivity.
func (e *SearchEngine) checkEmbedderHealth(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Component: "embedder",
		Timestamp: start,
		Details:   make(map[string]any),
	}

	// Try to embed a simple test string
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	testText := "health check test"
	embedding, err := e.embedder.Embed(testCtx, testText)

	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("embedder error: %v", err)
	} else if len(embedding) == 0 {
		check.Status = HealthStatusDegraded
		check.Message = "embedder returned empty embedding"
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "embedder healthy"
		check.Details["embedding_dimension"] = len(embedding)
	}

	check.Latency = time.Since(start)
	return check
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// HealthChecker is an interface for components that support health checking.
type HealthChecker interface {
	HealthCheck(ctx context.Context) HealthCheck
}

// Ensure interfaces are implemented.
var (
	_ HealthChecker = (*DocumentStore)(nil)
	_ HealthChecker = (*SearchEngine)(nil)
)
