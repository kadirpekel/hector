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

func NewChromaDatabaseProviderFromConfig(config *config.VectorStoreConfig) (DatabaseProvider, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required for Chroma")
	}

	scheme := "http"
	if config.EnableTLS != nil && *config.EnableTLS {
		scheme = "https"
	}

	port := config.Port
	if port == 0 {
		port = 8000 // Default Chroma port
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
			fmt.Printf("Warning: TLS certificate verification disabled for Chroma (insecure_skip_verify=true)\n")
		}
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport, // nil is fine, uses default transport
	}

	return &chromaDatabaseProvider{
		baseURL:    baseURL,
		apiKey:     config.APIKey,
		httpClient: httpClient,
		config:     config,
	}, nil
}

type chromaDatabaseProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     *config.VectorStoreConfig
}

func (db *chromaDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Chroma uses HTTP API
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	// Prepare documents and metadatas
	documents := []string{""}
	if content, ok := metadata["content"].(string); ok {
		documents[0] = content
	}

	metadatas := []map[string]interface{}{metadata}
	ids := []string{id}
	embeddings := [][]float64{vector64}

	payload := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/add", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
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

func (db *chromaDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(ctx, collection, queryVector, topK, nil)
}

func (db *chromaDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, queryVector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error) {
	vector64 := make([]float64, len(queryVector))
	for i, v := range queryVector {
		vector64[i] = float64(v)
	}

	queryEmbeddings := [][]float64{vector64}
	payload := map[string]interface{}{
		"query_embeddings": queryEmbeddings,
		"n_results":        topK,
	}

	if len(filter) > 0 {
		where := buildChromaWhere(filter)
		payload["where"] = where
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
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

	return convertChromaResults(result), nil
}

func (db *chromaDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]SearchResult, error) {
	// Chroma doesn't natively support hybrid search, use fallback
	if alpha >= 1.0 {
		return db.SearchWithFilter(ctx, collection, vector, topK, filter)
	}

	vectorResults, err := db.SearchWithFilter(ctx, collection, vector, topK*2, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	keywordResults := filterByKeywordsChroma(vectorResults, query, topK*2)
	fusedResults := reciprocalRankFusionChroma(vectorResults, keywordResults, alpha, topK)

	return fusedResults, nil
}

func buildChromaWhere(filter map[string]interface{}) map[string]interface{} {
	// Chroma uses simple where clause
	where := make(map[string]interface{})
	for k, v := range filter {
		where[k] = v
	}
	return where
}

func filterByKeywordsChroma(results []SearchResult, query string, limit int) []SearchResult {
	return filterByKeywords(results, query, limit)
}

func reciprocalRankFusionChroma(vectorResults, keywordResults []SearchResult, alpha float32, topK int) []SearchResult {
	return reciprocalRankFusion(vectorResults, keywordResults, alpha, topK)
}

func convertChromaResults(result map[string]interface{}) []SearchResult {
	if result == nil {
		return []SearchResult{}
	}

	// Chroma returns: { "ids": [[...]], "distances": [[...]], "documents": [[...]], "metadatas": [[...]] }
	ids, _ := result["ids"].([]interface{})
	if len(ids) == 0 {
		return []SearchResult{}
	}

	firstIds, _ := ids[0].([]interface{})
	distances, _ := result["distances"].([]interface{})
	firstDistances, _ := distances[0].([]interface{})
	documents, _ := result["documents"].([]interface{})
	firstDocs, _ := documents[0].([]interface{})
	metadatas, _ := result["metadatas"].([]interface{})
	firstMetas, _ := metadatas[0].([]interface{})

	results := make([]SearchResult, 0, len(firstIds))
	for i := 0; i < len(firstIds) && i < len(firstDistances); i++ {
		id := ""
		if idVal, ok := firstIds[i].(string); ok {
			id = idVal
		}

		score := float32(0)
		if distVal, ok := firstDistances[i].(float64); ok {
			score = float32(1.0 - distVal) // Convert distance to similarity
		}

		content := ""
		if i < len(firstDocs) && firstDocs[i] != nil {
			if docVal, ok := firstDocs[i].(string); ok {
				content = docVal
			}
		}

		metadata := make(map[string]interface{})
		if i < len(firstMetas) && firstMetas[i] != nil {
			if metaVal, ok := firstMetas[i].(map[string]interface{}); ok {
				metadata = metaVal
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

func (db *chromaDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	payload := map[string]interface{}{
		"ids": []string{id},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
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

func (db *chromaDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	where := buildChromaWhere(filter)
	payload := map[string]interface{}{
		"where": where,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
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

func (db *chromaDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	// Check if collection exists
	url := fmt.Sprintf("%s/api/v1/collections/%s", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
	}

	resp, err := db.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Collection already exists
	}

	// Create collection
	payload := map[string]interface{}{
		"name":          collection,
		"metadata":      map[string]interface{}{},
		"get_or_create": true,
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
		req.Header.Set("X-Api-Key", db.apiKey)
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

func (db *chromaDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", db.baseURL, collection)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if db.apiKey != "" {
		req.Header.Set("X-Api-Key", db.apiKey)
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

func (db *chromaDatabaseProvider) Close() error {
	return nil
}
