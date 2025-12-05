// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
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

// WeaviateConfig configures the Weaviate vector provider.
//
// Direct port from legacy pkg/databases/weaviate.go
type WeaviateConfig struct {
	// Host is the Weaviate server hostname.
	Host string `yaml:"host"`

	// Port is the Weaviate HTTP port (default: 8080).
	Port int `yaml:"port,omitempty"`

	// APIKey for authenticated access (optional).
	APIKey string `yaml:"api_key,omitempty"`

	// UseTLS enables HTTPS connections.
	UseTLS bool `yaml:"use_tls,omitempty"`
}

// WeaviateProvider implements Provider using Weaviate vector database.
//
// Direct port from legacy pkg/databases/weaviate.go
type WeaviateProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     WeaviateConfig
}

// NewWeaviateProvider creates a new Weaviate provider.
func NewWeaviateProvider(cfg WeaviateConfig) (*WeaviateProvider, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required for Weaviate")
	}

	scheme := "http"
	if cfg.UseTLS {
		scheme = "https"
	}

	port := cfg.Port
	if port == 0 {
		port = 8080
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, port)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &WeaviateProvider{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// Name returns the provider name.
func (p *WeaviateProvider) Name() string {
	return "weaviate"
}

// Upsert adds or updates a document with its vector.
func (p *WeaviateProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]any) error {
	// Prepare properties (metadata)
	properties := make(map[string]any)
	for k, v := range metadata {
		properties[k] = v
	}

	// Convert vector to []float64 for JSON
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	payload := map[string]any{
		"id":         id,
		"class":      collection,
		"properties": properties,
		"vector":     vector64,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects", p.baseURL)
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
		return fmt.Errorf("failed to upsert object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert object: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search finds the most similar vectors.
func (p *WeaviateProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]Result, error) {
	return p.SearchWithFilter(ctx, collection, vector, topK, nil)
}

// SearchWithFilter combines vector similarity with metadata filtering.
func (p *WeaviateProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]Result, error) {
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	query := map[string]any{
		"query": fmt.Sprintf(`
		{
			Get {
				%s {
					_additional {
						id
						certainty
						distance
					}
					content
				}
			}
		}`, collection),
		"nearVector": map[string]any{
			"vector": vector64,
		},
		"limit": topK,
	}

	if len(filter) > 0 {
		query["where"] = buildWeaviateWhereClause(filter)
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/v1/graphql", p.baseURL)
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

	return convertWeaviateResults(result, collection), nil
}

// Delete removes a document by ID.
func (p *WeaviateProvider) Delete(ctx context.Context, collection string, id string) error {
	url := fmt.Sprintf("%s/v1/objects/%s/%s", p.baseURL, collection, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete object: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteByFilter removes all documents matching the filter.
func (p *WeaviateProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]any) error {
	whereClause := buildWeaviateWhereClause(filter)
	if whereClause == nil {
		return fmt.Errorf("filter is required for delete by filter")
	}

	payload := map[string]any{
		"match": map[string]any{
			"class": collection,
			"where": whereClause,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/batch/objects", p.baseURL)
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

// CreateCollection creates a new class in Weaviate.
func (p *WeaviateProvider) CreateCollection(ctx context.Context, collection string, vectorDimension int) error {
	// Check if class exists
	url := fmt.Sprintf("%s/v1/schema/%s", p.baseURL, collection)
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
		return nil // Class already exists
	}

	// Create class schema
	classSchema := map[string]any{
		"class":      collection,
		"vectorizer": "none", // We provide vectors ourselves
		"properties": []map[string]any{
			{
				"name":     "content",
				"dataType": []string{"text"},
			},
		},
	}

	jsonData, err := json.Marshal(classSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	url = fmt.Sprintf("%s/v1/schema", p.baseURL)
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
		return fmt.Errorf("failed to create class: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create class: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteCollection removes a class from Weaviate.
func (p *WeaviateProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/v1/schema/%s", p.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete class: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete class: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the HTTP client.
func (p *WeaviateProvider) Close() error {
	return nil
}

// buildWeaviateWhereClause converts a filter map to Weaviate where clause.
func buildWeaviateWhereClause(filter map[string]any) map[string]any {
	if len(filter) == 0 {
		return nil
	}

	var conditions []map[string]any
	for key, value := range filter {
		conditions = append(conditions, map[string]any{
			"path":        []string{key},
			"operator":    "Equal",
			"valueString": fmt.Sprintf("%v", value),
		})
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return map[string]any{
		"operator": "And",
		"operands": conditions,
	}
}

// convertWeaviateResults converts Weaviate GraphQL response to our Result type.
func convertWeaviateResults(result map[string]any, collection string) []Result {
	if result == nil {
		return []Result{}
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return []Result{}
	}

	get, ok := data["Get"].(map[string]any)
	if !ok {
		return []Result{}
	}

	classData, ok := get[collection].([]any)
	if !ok {
		return []Result{}
	}

	results := make([]Result, 0, len(classData))
	for _, obj := range classData {
		objMap, ok := obj.(map[string]any)
		if !ok {
			continue
		}

		additional, _ := objMap["_additional"].(map[string]any)
		id := ""
		if idVal, ok := additional["id"].(string); ok {
			id = idVal
		}

		var score float32
		if certainty, ok := additional["certainty"].(float64); ok {
			score = float32(certainty)
		} else if distance, ok := additional["distance"].(float64); ok {
			score = float32(1.0 - distance)
		} else if scoreVal, ok := additional["score"].(float64); ok {
			score = float32(scoreVal)
		}

		content := ""
		if contentVal, ok := objMap["content"].(string); ok {
			content = contentVal
		}

		metadata := make(map[string]any)
		for k, v := range objMap {
			if k != "_additional" {
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

// Ensure WeaviateProvider implements Provider.
var _ Provider = (*WeaviateProvider)(nil)
