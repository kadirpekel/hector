package indexing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APISource implements DataSource for REST API endpoints
type APISource struct {
	client    *http.Client
	baseURL   string
	endpoints []APIEndpointConfig
	auth      *APIAuthConfig
}

// APIEndpointConfig defines an API endpoint to index
type APIEndpointConfig struct {
	Path           string            `yaml:"path"`            // API path (relative to baseURL)
	Method         string            `yaml:"method"`          // HTTP method (default: GET)
	Params         map[string]string `yaml:"params"`          // Query parameters
	Headers        map[string]string `yaml:"headers"`         // Additional headers
	Body           string            `yaml:"body"`            // Request body (for POST/PUT)
	IDField        string            `yaml:"id_field"`        // JSON field to use as document ID
	ContentField   string            `yaml:"content_field"`   // JSON field(s) to use as content (comma-separated or JSONPath)
	MetadataFields []string          `yaml:"metadata_fields"` // JSON fields to include as metadata
	UpdatedField   string            `yaml:"updated_field"`   // JSON field for last modified time
	Pagination     *PaginationConfig `yaml:"pagination"`      // Pagination configuration
	Transform      string            `yaml:"transform"`       // Optional JavaScript-like transform function (future)
}

// PaginationConfig defines how to handle paginated API responses
type PaginationConfig struct {
	Type      string `yaml:"type"`       // "offset", "cursor", "page", "link"
	PageParam string `yaml:"page_param"` // Query parameter name for page/offset
	SizeParam string `yaml:"size_param"` // Query parameter name for page size
	MaxPages  int    `yaml:"max_pages"`  // Maximum pages to fetch (0 = unlimited)
	PageSize  int    `yaml:"page_size"`  // Items per page
	NextField string `yaml:"next_field"` // JSON field containing next page URL/cursor
	DataField string `yaml:"data_field"` // JSON field containing array of items (if nested)
}

// APIAuthConfig defines authentication for API requests
type APIAuthConfig struct {
	Type   string            `yaml:"type"`   // "bearer", "basic", "apikey", "oauth2"
	Token  string            `yaml:"token"`  // Token/API key
	User   string            `yaml:"user"`   // Username (for basic auth)
	Pass   string            `yaml:"pass"`   // Password (for basic auth)
	Header string            `yaml:"header"` // Header name (for apikey type)
	Extra  map[string]string `yaml:"extra"`  // Additional auth parameters
}

// RateLimiter provides simple rate limiting for API requests
// Currently unused but reserved for future rate limiting implementation
type RateLimiter struct {
	_ struct{} // Reserved for future use
}

// NewAPISource creates a new REST API data source
func NewAPISource(baseURL string, endpoints []APIEndpointConfig, auth *APIAuthConfig) *APISource {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &APISource{
		client:    client,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		endpoints: endpoints,
		auth:      auth,
	}
}

func (a *APISource) Type() string {
	return "api"
}

func (a *APISource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(docChan)
		defer close(errChan)

		for _, endpoint := range a.endpoints {
			if err := a.indexEndpoint(ctx, endpoint, docChan, errChan); err != nil {
				select {
				case errChan <- fmt.Errorf("failed to index endpoint %s: %w", endpoint.Path, err):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return docChan, errChan
}

func (a *APISource) indexEndpoint(ctx context.Context, config APIEndpointConfig, docChan chan<- Document, errChan chan<- error) error {
	method := config.Method
	if method == "" {
		method = "GET"
	}

	// Handle pagination
	if config.Pagination != nil {
		return a.indexPaginatedEndpoint(ctx, config, docChan, errChan)
	}

	// Single request
	url := a.baseURL + config.Path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if len(config.Params) > 0 {
		q := req.URL.Query()
		for k, v := range config.Params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Add headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Add authentication
	a.addAuth(req)

	// Add body if present
	if config.Body != "" {
		req.Body = io.NopCloser(strings.NewReader(config.Body))
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract documents from response
	docs, err := a.extractDocuments(data, config)
	if err != nil {
		return fmt.Errorf("failed to extract documents: %w", err)
	}

	// Send documents
	for _, doc := range docs {
		select {
		case docChan <- doc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (a *APISource) indexPaginatedEndpoint(ctx context.Context, config APIEndpointConfig, docChan chan<- Document, errChan chan<- error) error {
	pagination := config.Pagination
	page := 1
	maxPages := pagination.MaxPages
	if maxPages == 0 {
		maxPages = 1000 // Safety limit
	}

	for page <= maxPages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		url := a.baseURL + config.Path
		req, err := http.NewRequestWithContext(ctx, config.Method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Add pagination parameters
		q := req.URL.Query()
		for k, v := range config.Params {
			q.Set(k, v)
		}

		switch pagination.Type {
		case "offset":
			offset := (page - 1) * pagination.PageSize
			q.Set(pagination.PageParam, fmt.Sprintf("%d", offset))
			q.Set(pagination.SizeParam, fmt.Sprintf("%d", pagination.PageSize))
		case "page":
			q.Set(pagination.PageParam, fmt.Sprintf("%d", page))
			q.Set(pagination.SizeParam, fmt.Sprintf("%d", pagination.PageSize))
		}

		req.URL.RawQuery = q.Encode()

		// Add headers and auth
		for k, v := range config.Headers {
			req.Header.Set(k, v)
		}
		a.addAuth(req)

		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
				// No more pages
				break
			}
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		// Extract data array if nested
		if pagination.DataField != "" {
			dataMap, ok := data.(map[string]interface{})
			if ok {
				data = dataMap[pagination.DataField]
			}
		}

		docs, err := a.extractDocuments(data, config)
		if err != nil {
			return fmt.Errorf("failed to extract documents: %w", err)
		}

		if len(docs) == 0 {
			// No more documents
			break
		}

		// Send documents
		for _, doc := range docs {
			select {
			case docChan <- doc:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Check for next page
		if pagination.NextField != "" {
			dataMap, ok := data.(map[string]interface{})
			if ok {
				nextVal := dataMap[pagination.NextField]
				if nextVal == nil || nextVal == "" {
					break
				}
			}
		}

		page++
	}

	return nil
}

func (a *APISource) extractDocuments(data interface{}, config APIEndpointConfig) ([]Document, error) {
	var items []interface{}

	// Handle array response
	if arr, ok := data.([]interface{}); ok {
		items = arr
	} else if obj, ok := data.(map[string]interface{}); ok {
		// Try to find array in object
		for _, v := range obj {
			if arr, ok := v.([]interface{}); ok {
				items = arr
				break
			}
		}
		// If no array found, treat object as single item
		if items == nil {
			items = []interface{}{obj}
		}
	} else {
		return nil, fmt.Errorf("unexpected response format")
	}

	docs := make([]Document, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract ID
		id := a.extractField(itemMap, config.IDField)
		if id == "" {
			id = fmt.Sprintf("%v", itemMap)
		}

		// Extract content
		content := a.extractContent(itemMap, config.ContentField)

		// Extract metadata
		metadata := make(map[string]interface{})
		metadata["endpoint"] = config.Path
		metadata["id"] = id
		for _, field := range config.MetadataFields {
			if val := a.extractFieldValue(itemMap, field); val != nil {
				metadata[field] = val
			}
		}

		// Extract last modified
		var lastModified time.Time
		if config.UpdatedField != "" {
			if val := a.extractFieldValue(itemMap, config.UpdatedField); val != nil {
				if str, ok := val.(string); ok {
					if t, err := time.Parse(time.RFC3339, str); err == nil {
						lastModified = t
					}
				}
			}
		}

		docs = append(docs, Document{
			ID:           fmt.Sprintf("api:%s:%s", config.Path, id),
			Content:      content,
			Metadata:     metadata,
			LastModified: lastModified,
			Size:         int64(len(content)),
			ShouldIndex:  true,
		})
	}

	return docs, nil
}

func (a *APISource) extractField(m map[string]interface{}, field string) string {
	if val := a.extractFieldValue(m, field); val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func (a *APISource) extractFieldValue(m map[string]interface{}, field string) interface{} {
	// Simple field extraction (could be enhanced with JSONPath)
	parts := strings.Split(field, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	return nil
}

func (a *APISource) extractContent(m map[string]interface{}, field string) string {
	if field == "" {
		// Default: try common fields
		for _, f := range []string{"content", "body", "text", "description", "summary"} {
			if val := a.extractFieldValue(m, f); val != nil {
				return fmt.Sprintf("%v", val)
			}
		}
		// Fallback: convert entire object to JSON
		if jsonBytes, err := json.Marshal(m); err == nil {
			return string(jsonBytes)
		}
		return ""
	}

	// Handle comma-separated fields
	fields := strings.Split(field, ",")
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if val := a.extractFieldValue(m, f); val != nil {
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}
	return strings.Join(parts, "\n\n")
}

func (a *APISource) addAuth(req *http.Request) {
	if a.auth == nil {
		return
	}

	switch a.auth.Type {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+a.auth.Token)
	case "basic":
		req.SetBasicAuth(a.auth.User, a.auth.Pass)
	case "apikey":
		headerName := a.auth.Header
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Header.Set(headerName, a.auth.Token)
	}
}

func (a *APISource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// Parse ID format: api:path:id
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid document ID format: %s", id)
	}

	// Find endpoint config
	var endpointConfig *APIEndpointConfig
	for _, cfg := range a.endpoints {
		if cfg.Path == parts[1] {
			endpointConfig = &cfg
			break
		}
	}
	if endpointConfig == nil {
		return nil, fmt.Errorf("endpoint %s not found in configuration", parts[1])
	}

	// Make request to fetch specific document
	// This would need endpoint-specific logic to construct the request
	// For now, return error indicating it's not implemented
	return nil, fmt.Errorf("ReadDocument not yet implemented for API sources")
}

func (a *APISource) SupportsIncrementalIndexing() bool {
	// API sources support incremental indexing if UpdatedField is configured
	for _, cfg := range a.endpoints {
		if cfg.UpdatedField != "" {
			return true
		}
	}
	return false
}

func (a *APISource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	doc, err := a.ReadDocument(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	return doc.LastModified, nil
}

func (a *APISource) Close() error {
	// Close HTTP client if needed
	return nil
}
