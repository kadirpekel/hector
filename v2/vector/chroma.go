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

package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// ChromaConfig configures the Chroma vector provider.
//
// Direct port from legacy pkg/databases/chroma.go
type ChromaConfig struct {
	// Host is the Chroma server hostname.
	Host string `yaml:"host"`

	// Port is the Chroma HTTP port (default: 8000).
	Port int `yaml:"port,omitempty"`

	// APIKey for authenticated access (optional).
	APIKey string `yaml:"api_key,omitempty"`

	// UseTLS enables HTTPS connections.
	UseTLS bool `yaml:"use_tls,omitempty"`
}

// ChromaProvider implements Provider using Chroma vector database.
//
// Direct port from legacy pkg/databases/chroma.go
type ChromaProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     ChromaConfig
}

// NewChromaProvider creates a new Chroma provider.
func NewChromaProvider(cfg ChromaConfig) (*ChromaProvider, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required for Chroma")
	}

	scheme := "http"
	if cfg.UseTLS {
		scheme = "https"
	}

	port := cfg.Port
	if port == 0 {
		port = 8000
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, port)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &ChromaProvider{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// Name returns the provider name.
func (p *ChromaProvider) Name() string {
	return "chroma"
}

// Upsert adds or updates a document with its vector.
func (p *ChromaProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]any) error {
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	// Prepare documents and metadatas
	documents := []string{""}
	if content, ok := metadata["content"].(string); ok {
		documents[0] = content
	}

	// Convert metadata to interface{}
	metadataInterface := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		metadataInterface[k] = v
	}

	payload := map[string]any{
		"ids":        []string{id},
		"embeddings": [][]float64{vector64},
		"documents":  documents,
		"metadatas":  []map[string]interface{}{metadataInterface},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/add", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upsert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search finds the most similar vectors.
func (p *ChromaProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]Result, error) {
	return p.SearchWithFilter(ctx, collection, vector, topK, nil)
}

// SearchWithFilter combines vector similarity with metadata filtering.
func (p *ChromaProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]Result, error) {
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	payload := map[string]any{
		"query_embeddings": [][]float64{vector64},
		"n_results":        topK,
	}

	if len(filter) > 0 {
		payload["where"] = filter
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return convertChromaResults(result), nil
}

// Delete removes a document by ID.
func (p *ChromaProvider) Delete(ctx context.Context, collection string, id string) error {
	payload := map[string]any{
		"ids": []string{id},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteByFilter removes all documents matching the filter.
func (p *ChromaProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]any) error {
	payload := map[string]any{
		"where": filter,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete by filter: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateCollection creates a new collection in Chroma.
func (p *ChromaProvider) CreateCollection(ctx context.Context, collection string, vectorDimension int) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/api/v1/collections/%s", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Collection already exists
	}

	// Create collection
	payload := map[string]any{
		"name":          collection,
		"metadata":      map[string]any{},
		"get_or_create": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url = fmt.Sprintf("%s/api/v1/collections", p.baseURL)
	req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err = p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteCollection removes a collection from Chroma.
func (p *ChromaProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("X-Api-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete collection: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the HTTP client.
func (p *ChromaProvider) Close() error {
	return nil
}

// convertChromaResults converts Chroma response to our Result type.
func convertChromaResults(result map[string]any) []Result {
	if result == nil {
		return []Result{}
	}

	// Chroma returns: { "ids": [[...]], "distances": [[...]], "documents": [[...]], "metadatas": [[...]] }
	ids, _ := result["ids"].([]any)
	if len(ids) == 0 {
		return []Result{}
	}

	firstIds, _ := ids[0].([]any)
	distances, _ := result["distances"].([]any)
	var firstDistances []any
	if len(distances) > 0 {
		firstDistances, _ = distances[0].([]any)
	}
	documents, _ := result["documents"].([]any)
	var firstDocs []any
	if len(documents) > 0 {
		firstDocs, _ = documents[0].([]any)
	}
	metadatas, _ := result["metadatas"].([]any)
	var firstMetas []any
	if len(metadatas) > 0 {
		firstMetas, _ = metadatas[0].([]any)
	}

	results := make([]Result, 0, len(firstIds))
	for i := 0; i < len(firstIds); i++ {
		id := ""
		if idVal, ok := firstIds[i].(string); ok {
			id = idVal
		}

		var score float32
		if i < len(firstDistances) {
			if distVal, ok := firstDistances[i].(float64); ok {
				score = float32(1.0 - distVal) // Convert distance to similarity
			}
		}

		content := ""
		if i < len(firstDocs) && firstDocs[i] != nil {
			if docVal, ok := firstDocs[i].(string); ok {
				content = docVal
			}
		}

		metadata := make(map[string]any)
		if i < len(firstMetas) && firstMetas[i] != nil {
			if metaVal, ok := firstMetas[i].(map[string]any); ok {
				metadata = metaVal
			}
		}

		results = append(results, Result{
			ID:       id,
			Content:  content,
			Score:    score,
			Metadata: metadata,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// Ensure ChromaProvider implements Provider.
var _ Provider = (*ChromaProvider)(nil)
