package databases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

func NewWeaviateDatabaseProviderFromConfig(config *config.VectorStoreConfig) (DatabaseProvider, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required for Weaviate")
	}

	scheme := "http"
	if config.EnableTLS != nil && *config.EnableTLS {
		scheme = "https"
	}

	port := config.Port
	if port == 0 {
		port = 8080 // Default Weaviate port
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, config.Host, port)

	// Configure TLS if needed
	var transport *http.Transport
	if scheme == "https" && (config.InsecureSkipVerify != nil && *config.InsecureSkipVerify || config.CACertificate != "") {
		tlsConfig := &httpclient.TLSConfig{
			InsecureSkipVerify: config.InsecureSkipVerify != nil && *config.InsecureSkipVerify,
			CACertificate:      config.CACertificate,
		}
		var err error
		transport, err = httpclient.ConfigureTLS(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		if tlsConfig.InsecureSkipVerify {
			fmt.Printf("Warning: TLS certificate verification disabled for Weaviate (insecure_skip_verify=true)\n")
		}
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport, // nil is fine, uses default transport
	}

	return &weaviateDatabaseProvider{
		baseURL:    baseURL,
		apiKey:     config.APIKey,
		httpClient: httpClient,
		config:     config,
	}, nil
}

type weaviateDatabaseProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     *config.VectorStoreConfig
}

func (db *weaviateDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Prepare properties (metadata)
	properties := make(map[string]interface{})
	for k, v := range metadata {
		properties[k] = v
	}

	// Convert vector to []float64 for JSON
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	// Build request payload
	payload := map[string]interface{}{
		"id":         id,
		"class":      collection,
		"properties": properties,
		"vector":     vector64,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Make HTTP request to Weaviate REST API
	url := fmt.Sprintf("%s/v1/objects", db.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
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

func (db *weaviateDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(ctx, collection, queryVector, topK, nil)
}

func (db *weaviateDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, queryVector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error) {
	// Convert vector to []float64
	vector64 := make([]float64, len(queryVector))
	for i, v := range queryVector {
		vector64[i] = float64(v)
	}

	// Build GraphQL query
	query := map[string]interface{}{
		"query": fmt.Sprintf(`
		{
			Get {
				%s {
					_additional {
						id
						certainty
						distance
					}
					*
				}
			}
		}`, collection),
		"nearVector": map[string]interface{}{
			"vector": vector64,
		},
		"limit": topK,
	}

	// Add where filter if provided
	if len(filter) > 0 {
		whereClause := buildWeaviateWhereClause(filter)
		query["where"] = whereClause
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Execute GraphQL query
	url := fmt.Sprintf("%s/v1/graphql", db.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return convertWeaviateResults(result, collection), nil
}

func (db *weaviateDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]SearchResult, error) {
	// Weaviate supports hybrid search natively via GraphQL
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	// Build hybrid GraphQL query
	graphqlQuery := fmt.Sprintf(`
		{
			Get {
				%s(
					hybrid: {
						query: "%s"
						vector: %s
						alpha: %f
					}
					limit: %d
				) {
					_additional {
						id
						certainty
						distance
						score
					}
					*
				}
			}
		}`, collection, query, vectorToJSONArray(vector64), alpha, topK)

	payload := map[string]interface{}{
		"query": graphqlQuery,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/v1/graphql", db.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hybrid search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return convertWeaviateResults(result, collection), nil
}

func buildWeaviateWhereClause(filter map[string]interface{}) map[string]interface{} {
	if len(filter) == 0 {
		return nil
	}

	// Build simple where clause (equality for now)
	var conditions []map[string]interface{}
	for key, value := range filter {
		conditions = append(conditions, map[string]interface{}{
			"path":        []string{key},
			"operator":    "Equal",
			"valueString": fmt.Sprintf("%v", value),
		})
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return map[string]interface{}{
		"operator": "And",
		"operands": conditions,
	}
}

func vectorToJSONArray(vector []float64) string {
	jsonBytes, _ := json.Marshal(vector)
	return string(jsonBytes)
}

func convertWeaviateResults(result map[string]interface{}, collection string) []SearchResult {
	if result == nil {
		return []SearchResult{}
	}

	// Extract results from GraphQL response
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return []SearchResult{}
	}

	get, ok := data["Get"].(map[string]interface{})
	if !ok {
		return []SearchResult{}
	}

	classData, ok := get[collection].([]interface{})
	if !ok {
		return []SearchResult{}
	}

	results := make([]SearchResult, 0, len(classData))
	for _, obj := range classData {
		objMap, ok := obj.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract ID from _additional
		additional, _ := objMap["_additional"].(map[string]interface{})
		id := ""
		if idVal, ok := additional["id"].(string); ok {
			id = idVal
		}

		// Extract score (certainty, distance, or score)
		var score float32
		if certainty, ok := additional["certainty"].(float64); ok {
			score = float32(certainty)
		} else if distance, ok := additional["distance"].(float64); ok {
			// Convert distance to similarity score
			score = float32(1.0 - distance)
		} else if scoreVal, ok := additional["score"].(float64); ok {
			score = float32(scoreVal)
		}

		// Extract content from properties
		content := ""
		if contentVal, ok := objMap["content"].(string); ok {
			content = contentVal
		}

		// Extract all properties as metadata
		metadata := make(map[string]interface{})
		for k, v := range objMap {
			if k != "_additional" {
				metadata[k] = v
			}
		}

		results = append(results, SearchResult{
			ID:       id,
			Content:  content,
			Score:    score,
			Metadata: metadata,
		})
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func (db *weaviateDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	url := fmt.Sprintf("%s/v1/objects/%s/%s", db.baseURL, collection, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
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

func (db *weaviateDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	whereClause := buildWeaviateWhereClause(filter)
	if whereClause == nil {
		return fmt.Errorf("filter is required for delete by filter")
	}

	payload := map[string]interface{}{
		"match": map[string]interface{}{
			"class": collection,
			"where": whereClause,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/batch/objects", db.baseURL)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
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

func (db *weaviateDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	// Check if class exists
	url := fmt.Sprintf("%s/v1/schema/%s", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Class already exists
	}

	// Create class schema
	classSchema := map[string]interface{}{
		"class":      collection,
		"vectorizer": "none", // We provide vectors ourselves
		"properties": []map[string]interface{}{
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

	url = fmt.Sprintf("%s/v1/schema", db.baseURL)
	req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err = db.httpClient.Do(req)
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

func (db *weaviateDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/v1/schema/%s", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
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

func (db *weaviateDatabaseProvider) Close() error {
	// Weaviate client doesn't have explicit close
	return nil
}
