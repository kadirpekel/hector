package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/kadirpekel/hector/config"
	hectorcontext "github.com/kadirpekel/hector/context"
)

// SearchTool provides search capabilities across configured document stores
type SearchTool struct {
	config          *config.SearchToolConfig
	availableStores []string // Names of document stores this tool can access
}

// SearchRequest represents a search query from an agent
type SearchRequest struct {
	Query    string            `json:"query"`
	Type     string            `json:"type"`     // "content", "file", "function", "struct"
	Stores   []string          `json:"stores"`   // Which document stores to search, empty = all
	Language string            `json:"language"` // Filter by language: "go", "yaml", "markdown"
	Limit    int               `json:"limit"`    // Max results, default 10
	Context  map[string]string `json:"context"`  // Additional search context
}

// DocumentSearchResult represents a single search result from document stores
type DocumentSearchResult struct {
	DocumentID string            `json:"document_id"`
	StoreName  string            `json:"store_name"`
	FilePath   string            `json:"file_path"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`  // Relevant content snippet
	Type       string            `json:"type"`     // Document type
	Language   string            `json:"language"` // Programming language
	Score      float64           `json:"score"`    // Relevance score
	LineNumber int               `json:"line_number,omitempty"`
	MatchType  string            `json:"match_type"` // "title", "content", "function", "struct"
	Metadata   map[string]string `json:"metadata"`
}

// SearchResponse contains search results and metadata
type SearchResponse struct {
	Results     []DocumentSearchResult `json:"results"`
	Total       int                    `json:"total"`
	Query       string                 `json:"query"`
	Duration    time.Duration          `json:"duration"`
	StoresUsed  []string               `json:"stores_used"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// NewSearchTool creates a new search tool with configuration
func NewSearchTool(searchConfig *config.SearchToolConfig) *SearchTool {
	if searchConfig == nil {
		// Default configuration
		searchConfig = &config.SearchToolConfig{
			DocumentStores:     []string{}, // Will use all available stores
			DefaultLimit:       10,
			MaxLimit:           50,
			EnabledSearchTypes: []string{"content", "file", "function", "struct"},
		}
	}

	// Set defaults
	if searchConfig.DefaultLimit == 0 {
		searchConfig.DefaultLimit = 10
	}
	if searchConfig.MaxLimit == 0 {
		searchConfig.MaxLimit = 50
	}
	if len(searchConfig.EnabledSearchTypes) == 0 {
		searchConfig.EnabledSearchTypes = []string{"content", "file", "function", "struct"}
	}

	return &SearchTool{
		config:          searchConfig,
		availableStores: searchConfig.DocumentStores,
	}
}

// NewSearchToolWithConfig creates a search tool from a ToolDefinition configuration
func NewSearchToolWithConfig(toolDef config.ToolDefinition) (*SearchTool, error) {
	// Convert the generic config map to SearchToolConfig
	var searchConfig *config.SearchToolConfig
	if toolDef.Config != nil {
		searchConfig = &config.SearchToolConfig{}
		// Map the common fields from map[string]interface{}
		if docStores, ok := toolDef.Config["document_stores"].([]interface{}); ok {
			stores := make([]string, len(docStores))
			for i, store := range docStores {
				if storeStr, ok := store.(string); ok {
					stores[i] = storeStr
				}
			}
			searchConfig.DocumentStores = stores
		}
		if defaultLimit, ok := toolDef.Config["default_limit"].(int); ok {
			searchConfig.DefaultLimit = defaultLimit
		}
		if maxLimit, ok := toolDef.Config["max_limit"].(int); ok {
			searchConfig.MaxLimit = maxLimit
		}
		if maxResults, ok := toolDef.Config["max_results"].(int); ok {
			searchConfig.MaxResults = maxResults
		}
		if enabledTypes, ok := toolDef.Config["enabled_search_types"].([]interface{}); ok {
			types := make([]string, len(enabledTypes))
			for i, typ := range enabledTypes {
				if typStr, ok := typ.(string); ok {
					types[i] = typStr
				}
			}
			searchConfig.EnabledSearchTypes = types
		}
	}

	// Apply defaults if config is nil
	if searchConfig == nil {
		searchConfig = &config.SearchToolConfig{}
	}
	searchConfig.SetDefaults()

	return NewSearchTool(searchConfig), nil
}

// getAvailableStores returns the document stores this tool can access
func (t *SearchTool) getAvailableStores() map[string]*hectorcontext.DocumentStore {
	stores := make(map[string]*hectorcontext.DocumentStore)

	// If no specific stores configured, use all available
	if len(t.availableStores) == 0 {
		storeNames := hectorcontext.ListDocumentStoresFromRegistry()
		for _, name := range storeNames {
			if store, exists := hectorcontext.GetDocumentStoreFromRegistry(name); exists {
				stores[name] = store
			}
		}
	} else {
		// Use only configured stores
		for _, name := range t.availableStores {
			if store, exists := hectorcontext.GetDocumentStoreFromRegistry(name); exists {
				stores[name] = store
			}
		}
	}

	return stores
}

// performSearch performs a search across document stores (internal method)
func (t *SearchTool) performSearch(ctx context.Context, req SearchRequest) (string, error) {
	start := time.Now()

	// Validate and set limits
	if req.Limit == 0 {
		req.Limit = t.config.DefaultLimit
	}
	if req.Limit > t.config.MaxLimit {
		req.Limit = t.config.MaxLimit
	}

	// Validate search type is enabled
	if !t.isSearchTypeEnabled(req.Type) {
		return t.createErrorResponse(fmt.Sprintf("Search type '%s' is not enabled", req.Type))
	}

	// Get available stores
	availableStores := t.getAvailableStores()
	if len(availableStores) == 0 {
		return t.createErrorResponse("No document stores available")
	}

	// Determine which stores to search
	storesToSearch := t.getStoresToSearch(req.Stores, availableStores)
	if len(storesToSearch) == 0 {
		return t.createErrorResponse("No matching document stores found")
	}

	// Perform search across selected stores
	var allResults []DocumentSearchResult
	var storesUsed []string

	for _, storeName := range storesToSearch {
		store, exists := hectorcontext.GetDocumentStoreFromRegistry(storeName)
		if !exists {
			fmt.Printf("Warning: Store %s not found in registry\n", storeName)
			continue
		}
		results, err := t.searchInStore(ctx, store, storeName, req)
		if err != nil {
			fmt.Printf("Warning: Search in store %s failed: %v\n", storeName, err)
			continue
		}

		allResults = append(allResults, results...)
		storesUsed = append(storesUsed, storeName)
	}

	// Sort results by score
	t.sortResultsByScore(allResults)

	// Limit results
	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}

	// Create response
	response := SearchResponse{
		Results:    allResults,
		Total:      len(allResults),
		Query:      req.Query,
		Duration:   time.Since(start),
		StoresUsed: storesUsed,
	}

	// Add suggestions if no results found
	if len(allResults) == 0 {
		response.Suggestions = t.generateSuggestions(req)
	}

	// Convert to JSON
	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal search response: %w", err)
	}

	return string(responseJSON), nil
}

// searchInStore searches within a specific document store
func (t *SearchTool) searchInStore(ctx context.Context, store *hectorcontext.DocumentStore, storeName string, req SearchRequest) ([]DocumentSearchResult, error) {
	var results []DocumentSearchResult

	// Use the document store's search method (vector DB backend)
	searchResults, err := store.Search(ctx, req.Query, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("store search failed: %w", err)
	}

	// Convert SearchResult to DocumentSearchResult
	for _, result := range searchResults {
		// Extract metadata
		language := ""
		filePath := ""
		title := ""
		docType := ""

		if result.Metadata != nil {
			if lang, ok := result.Metadata["language"].(string); ok {
				language = lang
			}
			if path, ok := result.Metadata["path"].(string); ok {
				filePath = path
			}
			if t, ok := result.Metadata["title"].(string); ok {
				title = t
			}
			if dt, ok := result.Metadata["type"].(string); ok {
				docType = dt
			}
		}

		// Apply language filter if specified
		if req.Language != "" && language != req.Language {
			continue
		}

		// Get content from metadata or use empty string
		content := ""
		if result.Metadata != nil {
			if c, ok := result.Metadata["content"].(string); ok {
				content = c[:minInt(len(c), 200)] // First 200 chars as preview
			}
		}

		documentResult := DocumentSearchResult{
			DocumentID: result.ID,
			StoreName:  storeName,
			FilePath:   filePath,
			Title:      title,
			Content:    content,
			Type:       docType,
			Language:   language,
			Score:      float64(result.Score),
			MatchType:  req.Type,
			Metadata:   make(map[string]string),
		}

		// Convert metadata to string map
		if result.Metadata != nil {
			for k, v := range result.Metadata {
				if str, ok := v.(string); ok {
					documentResult.Metadata[k] = str
				}
			}
		}

		results = append(results, documentResult)
	}

	return results, nil
}

// Helper methods

// isSearchTypeEnabled checks if a search type is enabled
func (t *SearchTool) isSearchTypeEnabled(searchType string) bool {
	for _, enabled := range t.config.EnabledSearchTypes {
		if enabled == searchType {
			return true
		}
	}
	return false
}

// getStoresToSearch determines which stores to search based on request and availability
func (t *SearchTool) getStoresToSearch(requestedStores []string, availableStores map[string]*hectorcontext.DocumentStore) []string {
	if len(requestedStores) == 0 {
		// Return all available stores
		var allStores []string
		for name := range availableStores {
			allStores = append(allStores, name)
		}
		return allStores
	}

	// Filter requested stores to only include available ones
	var validStores []string
	for _, name := range requestedStores {
		if _, exists := availableStores[name]; exists {
			validStores = append(validStores, name)
		}
	}

	return validStores
}

// sortResultsByScore sorts results by score in descending order using quicksort
func (t *SearchTool) sortResultsByScore(results []DocumentSearchResult) {
	if len(results) <= 1 {
		return
	}

	// Use Go's built-in sort for better performance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// generateSuggestions generates helpful suggestions when no results are found
func (t *SearchTool) generateSuggestions(req SearchRequest) []string {
	suggestions := []string{
		"Try a more specific search term",
		"Check if the document store is indexed",
		"Use 'file' type to search filenames",
		"Use 'function' type to find function definitions",
		"Use 'struct' type to find struct definitions",
	}

	if req.Language == "" {
		suggestions = append(suggestions, "Try specifying a language filter (go, yaml, markdown)")
	}

	if len(req.Stores) == 0 {
		suggestions = append(suggestions, "Try specifying which document stores to search")
	}

	return suggestions
}

// createErrorResponse creates a standardized error response
func (t *SearchTool) createErrorResponse(message string) (string, error) {
	response := SearchResponse{
		Results:     []DocumentSearchResult{},
		Total:       0,
		Query:       "",
		Duration:    0,
		StoresUsed:  []string{},
		Suggestions: []string{message},
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(responseJSON), nil
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Tool interface implementation

// GetInfo returns tool information for the Tool interface
func (t *SearchTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "search",
		Description: "Search across configured document stores for files, content, functions, or structs",
		Parameters: []ToolParameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query text",
				Required:    true,
			},
			{
				Name:        "type",
				Type:        "string",
				Description: "Search type: content, file, function, struct",
				Required:    false,
				Default:     "content",
				Enum:        []string{"content", "file", "function", "struct"},
			},
			{
				Name:        "stores",
				Type:        "array",
				Description: "Document stores to search (empty = all)",
				Required:    false,
			},
			{
				Name:        "language",
				Type:        "string",
				Description: "Filter by programming language",
				Required:    false,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Maximum number of results",
				Required:    false,
				Default:     10,
			},
		},
		ServerURL: "local",
	}
}

// GetName returns the tool name
func (t *SearchTool) GetName() string {
	return "search"
}

// GetDescription returns the tool description
func (t *SearchTool) GetDescription() string {
	return "Search across configured document stores for files, content, functions, or structs"
}

// Execute executes the search tool with structured arguments (Tool interface)
func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Extract parameters from args
	query, _ := args["query"].(string)
	if query == "" {
		return ToolResult{
			Success:       false,
			Error:         "query parameter is required",
			ToolName:      "search",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("query parameter is required")
	}

	// Build search request
	req := SearchRequest{
		Query: query,
		Type:  getStringWithDefault(args, "type", "content"),
		Limit: getIntWithDefault(args, "limit", 10),
	}

	// Handle stores parameter (can be array or single string)
	if stores, ok := args["stores"]; ok {
		switch v := stores.(type) {
		case []interface{}:
			for _, store := range v {
				if s, ok := store.(string); ok {
					req.Stores = append(req.Stores, s)
				}
			}
		case []string:
			req.Stores = v
		case string:
			if v != "" {
				req.Stores = []string{v}
			}
		}
	}

	if language, ok := args["language"].(string); ok {
		req.Language = language
	}

	// Execute the search using the existing method
	content, err := t.performSearch(ctx, req)
	if err != nil {
		return ToolResult{
			Success:       false,
			Error:         err.Error(),
			ToolName:      "search",
			ExecutionTime: time.Since(start),
		}, err
	}

	return ToolResult{
		Success:       true,
		Content:       content,
		ToolName:      "search",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"repository": "local",
			"tool_type":  "search",
		},
	}, nil
}

// Helper functions for parameter extraction
func getStringWithDefault(args map[string]interface{}, key, defaultValue string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntWithDefault(args map[string]interface{}, key string, defaultValue int) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	if val, ok := args[key].(int); ok {
		return val
	}
	return defaultValue
}
