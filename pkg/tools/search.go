package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

type SearchTool struct {
	config          *config.SearchToolConfig
	availableStores []string
}

type SearchRequest struct {
	Query    string            `json:"query"`
	Type     string            `json:"type"`
	Stores   []string          `json:"stores"`
	Language string            `json:"language"`
	Limit    int               `json:"limit"`
	Context  map[string]string `json:"context"`
}

type DocumentSearchResult struct {
	DocumentID string            `json:"document_id"`
	StoreName  string            `json:"store_name"`
	FilePath   string            `json:"file_path"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Type       string            `json:"type"`
	Language   string            `json:"language"`
	Score      float64           `json:"score"`
	StartLine  int               `json:"start_line,omitempty"`
	EndLine    int               `json:"end_line,omitempty"`
	LineNumber int               `json:"line_number,omitempty"`
	MatchType  string            `json:"match_type"`
	Metadata   map[string]string `json:"metadata"`
}

type SearchResponse struct {
	Results     []DocumentSearchResult `json:"results"`
	Total       int                    `json:"total"`
	Query       string                 `json:"query"`
	Duration    time.Duration          `json:"duration"`
	StoresUsed  []string               `json:"stores_used"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

func NewSearchTool(searchConfig *config.SearchToolConfig) *SearchTool {
	if searchConfig == nil {

		searchConfig = &config.SearchToolConfig{
			DocumentStores:     []string{},
			DefaultLimit:       10,
			MaxLimit:           50,
			EnabledSearchTypes: []string{"content", "file", "function", "struct"},
		}
	}

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

func NewSearchToolWithConfig(name string, toolConfig *config.ToolConfig) (*SearchTool, error) {
	if toolConfig == nil {
		return nil, fmt.Errorf("tool config is required")
	}

	searchConfig := &config.SearchToolConfig{
		DocumentStores:     toolConfig.DocumentStores,
		DefaultLimit:       toolConfig.DefaultLimit,
		MaxLimit:           toolConfig.MaxLimit,
		EnabledSearchTypes: toolConfig.EnabledSearchTypes,
	}

	searchConfig.SetDefaults()

	return NewSearchTool(searchConfig), nil
}

func (t *SearchTool) getAvailableStores() map[string]*hectorcontext.DocumentStore {
	stores := make(map[string]*hectorcontext.DocumentStore)

	if len(t.availableStores) == 0 {
		storeNames := hectorcontext.ListDocumentStoresFromRegistry()
		for _, name := range storeNames {
			if store, exists := hectorcontext.GetDocumentStoreFromRegistry(name); exists {
				stores[name] = store
			}
		}
	} else {

		for _, name := range t.availableStores {
			if store, exists := hectorcontext.GetDocumentStoreFromRegistry(name); exists {
				stores[name] = store
			}
		}
	}

	return stores
}

func (t *SearchTool) performSearch(ctx context.Context, req SearchRequest) (string, error) {
	start := time.Now()

	if req.Limit == 0 {
		req.Limit = t.config.DefaultLimit
	}
	if req.Limit > t.config.MaxLimit {
		req.Limit = t.config.MaxLimit
	}

	if !t.isSearchTypeEnabled(req.Type) {
		return t.createErrorResponse(fmt.Sprintf("Search type '%s' is not enabled", req.Type))
	}

	availableStores := t.getAvailableStores()
	if len(availableStores) == 0 {
		return t.createErrorResponse("No document stores available")
	}

	storesToSearch := t.getStoresToSearch(req.Stores, availableStores)
	if len(storesToSearch) == 0 {
		return t.createErrorResponse("No matching document stores found")
	}

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
			fmt.Printf("Error: Search in store %s failed: %v\n", storeName, err)
			continue
		}

		allResults = append(allResults, results...)
		storesUsed = append(storesUsed, storeName)
	}

	t.sortResultsByScore(allResults)

	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}

	response := SearchResponse{
		Results:    allResults,
		Total:      len(allResults),
		Query:      req.Query,
		Duration:   time.Since(start),
		StoresUsed: storesUsed,
	}

	if len(allResults) == 0 {
		response.Suggestions = t.generateSuggestions(req)
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal search response: %w", err)
	}

	return string(responseJSON), nil
}

func (t *SearchTool) searchInStore(ctx context.Context, store *hectorcontext.DocumentStore, storeName string, req SearchRequest) ([]DocumentSearchResult, error) {
	var results []DocumentSearchResult

	searchResults, err := store.Search(ctx, req.Query, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("store search failed: %w", err)
	}

	for _, result := range searchResults {

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

		if req.Language != "" && language != req.Language {
			continue
		}

		startLine := 0
		endLine := 0
		if result.Metadata != nil {
			if sl, ok := result.Metadata["start_line"].(float64); ok {
				startLine = int(sl)
			}
			if el, ok := result.Metadata["end_line"].(float64); ok {
				endLine = int(el)
			}
		}

		content := ""
		if result.Metadata != nil {
			if c, ok := result.Metadata["content"].(string); ok {
				rawContent := c
				wasTruncated := false

				if len(rawContent) > 2000 {
					rawContent = rawContent[:2000]
					wasTruncated = true
				}

				if startLine > 0 && endLine > 0 {
					header := fmt.Sprintf("📄 %s (lines %d-%d)\n%s",
						filePath,
						startLine,
						endLine,
						strings.Repeat("-", 60))

					if wasTruncated {
						content = fmt.Sprintf("%s\n%s\n\nWarning: TRUNCATED - Use: sed -n \"%d,%dp\" %s",
							header,
							rawContent,
							startLine,
							endLine,
							filePath)
					} else {
						content = fmt.Sprintf("%s\n%s", header, rawContent)
					}
				} else {
					content = rawContent
				}
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
			StartLine:  startLine,
			EndLine:    endLine,
			MatchType:  req.Type,
			Metadata:   make(map[string]string),
		}

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

func (t *SearchTool) isSearchTypeEnabled(searchType string) bool {
	for _, enabled := range t.config.EnabledSearchTypes {
		if enabled == searchType {
			return true
		}
	}
	return false
}

func (t *SearchTool) getStoresToSearch(requestedStores []string, availableStores map[string]*hectorcontext.DocumentStore) []string {
	if len(requestedStores) == 0 {

		var allStores []string
		for name := range availableStores {
			allStores = append(allStores, name)
		}
		return allStores
	}

	var validStores []string
	for _, name := range requestedStores {
		if _, exists := availableStores[name]; exists {
			validStores = append(validStores, name)
		}
	}

	return validStores
}

func (t *SearchTool) sortResultsByScore(results []DocumentSearchResult) {
	if len(results) <= 1 {
		return
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
				Items: map[string]interface{}{
					"type": "string",
				},
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

func (t *SearchTool) GetName() string {
	return "search"
}

func (t *SearchTool) GetDescription() string {
	return "Search across configured document stores for files, content, functions, or structs using semantic search. Results include line numbers for precise code references."
}

func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	query, _ := args["query"].(string)
	if query == "" {
		return ToolResult{
			Success:       false,
			Error:         "query parameter is required",
			ToolName:      "search",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("query parameter is required")
	}

	req := SearchRequest{
		Query: query,
		Type:  getStringWithDefault(args, "type", "content"),
		Limit: getIntWithDefault(args, "limit", 10),
	}

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
			"source":                  "local",
			"tool_type":               "search",
			"reflection_context_size": 2000,
		},
	}, nil
}

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
