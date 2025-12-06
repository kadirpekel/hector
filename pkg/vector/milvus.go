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

// MilvusConfig configures the Milvus vector provider.
//
// Direct port from legacy pkg/databases/milvus.go
type MilvusConfig struct {
	// Host is the Milvus server hostname.
	Host string `yaml:"host"`

	// Port is the Milvus HTTP port (default: 19530).
	Port int `yaml:"port,omitempty"`

	// APIKey for authenticated access (optional).
	APIKey string `yaml:"api_key,omitempty"`

	// UseTLS enables HTTPS connections.
	UseTLS bool `yaml:"use_tls,omitempty"`
}

// MilvusProvider implements Provider using Milvus vector database.
//
// Direct port from legacy pkg/databases/milvus.go
type MilvusProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     MilvusConfig
}

// NewMilvusProvider creates a new Milvus provider.
func NewMilvusProvider(cfg MilvusConfig) (*MilvusProvider, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required for Milvus")
	}

	scheme := "http"
	if cfg.UseTLS {
		scheme = "https"
	}

	port := cfg.Port
	if port == 0 {
		port = 19530
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, port)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &MilvusProvider{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// Name returns the provider name.
func (p *MilvusProvider) Name() string {
	return "milvus"
}

// Upsert adds or updates a document with its vector.
func (p *MilvusProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]any) error {
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	dataItem := map[string]any{
		"id":     id,
		"vector": vector64,
	}

	// Add metadata fields
	for k, v := range metadata {
		dataItem[k] = v
	}

	payload := map[string]any{
		"collection_name": collection,
		"data":            []map[string]any{dataItem},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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
func (p *MilvusProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]Result, error) {
	return p.SearchWithFilter(ctx, collection, vector, topK, nil)
}

// SearchWithFilter combines vector similarity with metadata filtering.
func (p *MilvusProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]Result, error) {
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	payload := map[string]any{
		"collection_name": collection,
		"vector":          vector64,
		"top_k":           topK,
		"metric_type":     "COSINE",
	}

	if len(filter) > 0 {
		payload["expr"] = buildMilvusFilter(filter)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/search", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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

	return convertMilvusResults(result), nil
}

// Delete removes a document by ID.
func (p *MilvusProvider) Delete(ctx context.Context, collection string, id string) error {
	payload := map[string]any{
		"collection_name": collection,
		"ids":             []string{id},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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
func (p *MilvusProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]any) error {
	expr := buildMilvusFilter(filter)
	if expr == "" {
		return fmt.Errorf("filter is required for delete by filter")
	}

	payload := map[string]any{
		"collection_name": collection,
		"expr":            expr,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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

// CreateCollection creates a new collection in Milvus.
func (p *MilvusProvider) CreateCollection(ctx context.Context, collection string, vectorDimension int) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/api/v1/collections/%s", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	resp, err := p.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Collection already exists
	}

	// Create collection
	payload := map[string]any{
		"collection_name": collection,
		"dimension":       vectorDimension,
		"metric_type":     "COSINE",
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
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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

// DeleteCollection removes a collection from Milvus.
func (p *MilvusProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
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
func (p *MilvusProvider) Close() error {
	return nil
}

// buildMilvusFilter converts a filter map to Milvus expression filter.
func buildMilvusFilter(filter map[string]any) string {
	if len(filter) == 0 {
		return ""
	}

	var conditions []string
	for key, value := range filter {
		conditions = append(conditions, fmt.Sprintf(`%s == "%v"`, key, value))
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return "(" + conditions[0] + " && " + conditions[1] + ")"
}

// convertMilvusResults converts Milvus response to our Result type.
func convertMilvusResults(result map[string]any) []Result {
	if result == nil {
		return []Result{}
	}

	resultsData, ok := result["results"].([]any)
	if !ok {
		return []Result{}
	}

	results := make([]Result, 0, len(resultsData))
	for _, item := range resultsData {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		id := ""
		if idVal, ok := itemMap["id"].(string); ok {
			id = idVal
		} else if idVal, ok := itemMap["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", idVal)
		}

		var score float32
		if scoreVal, ok := itemMap["distance"].(float64); ok {
			score = float32(1.0 - scoreVal) // Convert distance to similarity
		} else if scoreVal, ok := itemMap["score"].(float64); ok {
			score = float32(scoreVal)
		}

		content := ""
		if contentVal, ok := itemMap["content"].(string); ok {
			content = contentVal
		}

		metadata := make(map[string]any)
		for k, v := range itemMap {
			if k != "id" && k != "distance" && k != "score" && k != "vector" {
				metadata[k] = v
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

// Ensure MilvusProvider implements Provider.
var _ Provider = (*MilvusProvider)(nil)
