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

func NewMilvusDatabaseProviderFromConfig(config *config.VectorStoreConfig) (DatabaseProvider, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required for Milvus")
	}

	scheme := "http"
	if config.EnableTLS != nil && *config.EnableTLS {
		scheme = "https"
	}

	port := config.Port
	if port == 0 {
		port = 19530 // Default Milvus port
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
			fmt.Printf("Warning: TLS certificate verification disabled for Milvus (insecure_skip_verify=true)\n")
		}
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport, // nil is fine, uses default transport
	}

	return &milvusDatabaseProvider{
		baseURL:    baseURL,
		apiKey:     config.APIKey,
		httpClient: httpClient,
		config:     config,
	}, nil
}

type milvusDatabaseProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     *config.VectorStoreConfig
}

func (db *milvusDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Milvus uses gRPC primarily, but we'll use HTTP API for simplicity
	// Convert vector to []float64 for JSON
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	// Build payload for Milvus insert
	payload := map[string]interface{}{
		"collection_name": collection,
		"data": []map[string]interface{}{
			{
				"id":     id,
				"vector": vector64,
			},
		},
	}

	// Add metadata fields
	if len(metadata) > 0 {
		for k, v := range metadata {
			payload["data"].([]map[string]interface{})[0][k] = v
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", db.baseURL)
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
		return fmt.Errorf("failed to upsert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (db *milvusDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(ctx, collection, queryVector, topK, nil)
}

func (db *milvusDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, queryVector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error) {
	vector64 := make([]float64, len(queryVector))
	for i, v := range queryVector {
		vector64[i] = float64(v)
	}

	payload := map[string]interface{}{
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

	url := fmt.Sprintf("%s/api/v1/search", db.baseURL)
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

	return convertMilvusResults(result), nil
}

func (db *milvusDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]SearchResult, error) {
	// Milvus doesn't natively support hybrid search, use fallback approach
	if alpha >= 1.0 {
		return db.SearchWithFilter(ctx, collection, vector, topK, filter)
	}

	vectorResults, err := db.SearchWithFilter(ctx, collection, vector, topK*2, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	keywordResults := filterByKeywordsMilvus(vectorResults, query, topK*2)
	fusedResults := reciprocalRankFusionMilvus(vectorResults, keywordResults, alpha, topK)

	return fusedResults, nil
}

func buildMilvusFilter(filter map[string]interface{}) string {
	// Build Milvus expression filter (simplified)
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

func filterByKeywordsMilvus(results []SearchResult, query string, limit int) []SearchResult {
	return filterByKeywords(results, query, limit)
}

func reciprocalRankFusionMilvus(vectorResults, keywordResults []SearchResult, alpha float32, topK int) []SearchResult {
	return reciprocalRankFusion(vectorResults, keywordResults, alpha, topK)
}

func convertMilvusResults(result map[string]interface{}) []SearchResult {
	if result == nil {
		return []SearchResult{}
	}

	// Extract results from Milvus response
	resultsData, ok := result["results"].([]interface{})
	if !ok {
		return []SearchResult{}
	}

	results := make([]SearchResult, 0, len(resultsData))
	for _, item := range resultsData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := ""
		if idVal, ok := itemMap["id"].(string); ok {
			id = idVal
		} else if idVal, ok := itemMap["id"].(float64); ok {
			id = fmt.Sprintf("%.0f", idVal)
		}

		score := float32(0)
		if scoreVal, ok := itemMap["distance"].(float64); ok {
			score = float32(1.0 - scoreVal) // Convert distance to similarity
		} else if scoreVal, ok := itemMap["score"].(float64); ok {
			score = float32(scoreVal)
		}

		content := ""
		if contentVal, ok := itemMap["content"].(string); ok {
			content = contentVal
		}

		metadata := make(map[string]interface{})
		for k, v := range itemMap {
			if k != "id" && k != "distance" && k != "score" && k != "vector" {
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

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func (db *milvusDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	payload := map[string]interface{}{
		"collection_name": collection,
		"ids":             []string{id},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", db.baseURL)
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
		return fmt.Errorf("failed to delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (db *milvusDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	expr := buildMilvusFilter(filter)
	if expr == "" {
		return fmt.Errorf("filter is required for delete by filter")
	}

	payload := map[string]interface{}{
		"collection_name": collection,
		"expr":            expr,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/entities", db.baseURL)
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

func (db *milvusDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/api/v1/collections/%s", db.baseURL, collection)
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
		return nil // Collection already exists
	}

	// Create collection
	payload := map[string]interface{}{
		"collection_name": collection,
		"dimension":       vectorSize,
		"metric_type":     "COSINE",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url = fmt.Sprintf("%s/api/v1/collections", db.baseURL)
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
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (db *milvusDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", db.apiKey))
	}

	resp, err := db.httpClient.Do(req)
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

func (db *milvusDatabaseProvider) Close() error {
	return nil
}
