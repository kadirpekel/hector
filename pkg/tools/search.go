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
	"github.com/kadirpekel/hector/pkg/databases"
)

type SearchTool struct {
	config          *config.SearchToolConfig
	availableStores []string
}

type SearchRequest struct {
	Query    string            `json:"query"`
	Type     string            `json:"type"`
	Stores   []string          `json:"stores"` // Document stores to search (empty = all stores)
	Language string            `json:"language"`
	Limit    int               `json:"limit"`
	Context  map[string]string `json:"context"`
}

type DocumentSearchResult struct {
	DocumentID       string            `json:"document_id"`
	StoreName        string            `json:"store_name"`
	StoreDescription string            `json:"store_description,omitempty"`
	FilePath         string            `json:"file_path"`
	Title            string            `json:"title"`
	Content          string            `json:"content"`
	Type             string            `json:"type"`
	Language         string            `json:"language"`
	Score            float64           `json:"score"`
	StartLine        int               `json:"start_line,omitempty"`
	EndLine          int               `json:"end_line,omitempty"`
	LineNumber       int               `json:"line_number,omitempty"`
	MatchType        string            `json:"match_type"`
	Metadata         map[string]string `json:"metadata"`
}

type SearchResponse struct {
	Results     []DocumentSearchResult `json:"results"`
	Total       int                    `json:"total"`
	Query       string                 `json:"query"`
	Duration    time.Duration          `json:"duration"`
	StoresUsed  []string               `json:"stores_used"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// collectionSearchInfo holds information about a collection to search
type collectionSearchInfo struct {
	collectionName string                      // The collection name to search
	searchEngine   *hectorcontext.SearchEngine // The search engine to use
	storeNames     []string                    // Store names that use this collection (for filtering/attribution)
}

// GetID implements ParallelSearchTarget interface
func (c *collectionSearchInfo) GetID() string {
	return c.collectionName
}

// collectionSearchResult holds results from a single collection search
type collectionSearchResult struct {
	collectionName string
	results        []DocumentSearchResult
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

	// Add a timeout to prevent indefinite hanging (30 seconds max per search)
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if req.Limit == 0 {
		req.Limit = t.config.DefaultLimit
	}
	if req.Limit > t.config.MaxLimit {
		req.Limit = t.config.MaxLimit
	}

	if !t.isSearchTypeEnabled(req.Type) {
		return t.createErrorResponse(fmt.Sprintf("Search type '%s' is not enabled", req.Type))
	}

	var allResults []DocumentSearchResult
	var storesUsed []string

	// Get available stores
	availableStores := t.getAvailableStores()
	if len(availableStores) == 0 {
		return t.createErrorResponse("No document stores available")
	}

	// Determine which stores to search (if specified)
	storesToSearch := t.getStoresToSearch(req.Stores, availableStores)
	if len(storesToSearch) == 0 {
		return t.createErrorResponse("No matching document stores found")
	}

	// Build collection search info: group stores by their collections
	collectionsToSearch := t.buildCollectionSearchInfo(storesToSearch, availableStores)
	if len(collectionsToSearch) == 0 {
		return t.createErrorResponse("No collections found to search")
	}

	// Convert to pointers for ParallelSearch
	collectionTargets := make([]*collectionSearchInfo, len(collectionsToSearch))
	for i := range collectionsToSearch {
		collectionTargets[i] = &collectionsToSearch[i]
	}

	// Use shared parallel search function
	searchFunc := func(ctx context.Context, target hectorcontext.ParallelSearchTarget) (collectionSearchResult, error) {
		collInfo := target.(*collectionSearchInfo)

		// Search the collection
		// Note: Store-specific filtering happens in result processing based on metadata
		filter := map[string]interface{}{
			"collection": collInfo.collectionName,
		}

		searchResults, err := collInfo.searchEngine.SearchWithFilter(ctx, req.Query, req.Limit, filter)
		if err != nil {
			return collectionSearchResult{}, fmt.Errorf("search in collection %s failed: %w", collInfo.collectionName, err)
		}

		// Convert search results to document search results
		results := t.convertSearchResults(searchResults, collInfo.storeNames, req)

		return collectionSearchResult{
			collectionName: collInfo.collectionName,
			results:        results,
		}, nil
	}

	// Convert collectionSearchInfo to ParallelSearchTarget slice
	targets := make([]hectorcontext.ParallelSearchTarget, len(collectionTargets))
	for i, target := range collectionTargets {
		targets[i] = target
	}

	parallelResults, err := hectorcontext.ParallelSearch(searchCtx, targets, searchFunc)
	if err != nil {
		// Return partial results if context was cancelled
		return t.buildSearchResponse(allResults, storesUsed, req, start)
	}

	// Collect results from all collections
	seenStores := make(map[string]bool)
	for _, parallelResult := range parallelResults {
		if parallelResult.Error != nil {
			// Log error but continue with other results
			fmt.Printf("Error: Search in collection %s failed: %v\n", parallelResult.TargetID, parallelResult.Error)
			continue
		}

		result := parallelResult.Results
		allResults = append(allResults, result.results...)
		// Track unique stores used
		for _, res := range result.results {
			if res.StoreName != "" && !seenStores[res.StoreName] {
				storesUsed = append(storesUsed, res.StoreName)
				seenStores[res.StoreName] = true
			}
		}
	}

	return t.buildSearchResponse(allResults, storesUsed, req, start)
}

// buildCollectionSearchInfo groups stores by their collections
// Returns a map of collection name -> collectionSearchInfo
func (t *SearchTool) buildCollectionSearchInfo(storesToSearch []string, availableStores map[string]*hectorcontext.DocumentStore) []collectionSearchInfo {
	// Map: collection name -> collectionSearchInfo
	collectionMap := make(map[string]*collectionSearchInfo)

	for _, storeName := range storesToSearch {
		store, exists := availableStores[storeName]
		if !exists {
			continue
		}

		searchEngine := store.GetSearchEngine()
		if searchEngine == nil {
			continue
		}

		// Get collection name from store (uses Collection override if set, otherwise store name)
		collectionName := store.GetCollectionName()

		// Get or create collection info
		if info, exists := collectionMap[collectionName]; exists {
			// Collection already exists, add this store to it
			info.storeNames = append(info.storeNames, storeName)
		} else {
			// New collection
			collectionMap[collectionName] = &collectionSearchInfo{
				collectionName: collectionName,
				searchEngine:   searchEngine,
				storeNames:     []string{storeName},
			}
		}
	}

	// Convert map to slice
	result := make([]collectionSearchInfo, 0, len(collectionMap))
	for _, info := range collectionMap {
		result = append(result, *info)
	}

	return result
}

// convertSearchResults converts database search results to document search results
// Filters by storeNames if provided, otherwise includes all results
func (t *SearchTool) convertSearchResults(searchResults []databases.SearchResult, allowedStoreNames []string, req SearchRequest) []DocumentSearchResult {
	var results []DocumentSearchResult
	allowedStoresMap := make(map[string]bool)
	for _, name := range allowedStoreNames {
		allowedStoresMap[name] = true
	}

	for _, result := range searchResults {
		// Extract metadata
		language := ""
		filePath := ""
		title := ""
		docType := ""
		storeName := ""

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
			// Get store_name from metadata (this is how we know which store the result belongs to)
			if sn, ok := result.Metadata["store_name"].(string); ok {
				storeName = sn
			}
		}

		// Filter by store_name if specific stores were requested
		if len(allowedStoreNames) > 0 {
			if storeName == "" || !allowedStoresMap[storeName] {
				continue // Skip results not from requested stores
			}
		}

		// Filter by language if specified
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

		// Build store description
		storeDescription := ""
		if storeName != "" {
			if store, exists := hectorcontext.GetDocumentStoreFromRegistry(storeName); exists {
				storeDescription = t.buildStoreDescriptionFromStoreSafe(store, storeName)
			}
		}

		metadata := make(map[string]string)
		if result.Metadata != nil {
			for k, v := range result.Metadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
		}

		documentResult := DocumentSearchResult{
			DocumentID:       result.ID,
			StoreName:        storeName,
			StoreDescription: storeDescription,
			FilePath:         filePath,
			Title:            title,
			Content:          content,
			Type:             docType,
			Language:         language,
			Score:            float64(result.Score),
			StartLine:        startLine,
			EndLine:          endLine,
			MatchType:        req.Type,
			Metadata:         metadata,
		}

		results = append(results, documentResult)
	}

	return results
}

// buildSearchResponse builds the final search response with sorting and limiting
func (t *SearchTool) buildSearchResponse(allResults []DocumentSearchResult, storesUsed []string, req SearchRequest, start time.Time) (string, error) {
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
	// Get available stores to include in description
	storeNames := hectorcontext.ListDocumentStoresFromRegistry()

	description := "Search across configured document stores for files, content, functions, or structs"
	if len(storeNames) > 0 {
		description += fmt.Sprintf(". Available stores: %s", strings.Join(storeNames, ", "))
	}

	return ToolInfo{
		Name:        "search",
		Description: description,
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

// buildStoreDescriptionFromStore builds a description from an existing store reference
// This is more efficient when you already have the store object
func (t *SearchTool) buildStoreDescriptionFromStore(store *hectorcontext.DocumentStore, storeName string) string {
	if store == nil {
		return ""
	}

	// Get store config and status
	config := store.GetConfig()
	status := store.GetStatus()

	// Build description parts
	var descParts []string

	// Add source type if available
	if config != nil && config.Source != "" {
		descParts = append(descParts, fmt.Sprintf("source: %s", config.Source))
	}

	// Add source path if available
	if status != nil && status.SourcePath != "" {
		descParts = append(descParts, fmt.Sprintf("path: %s", status.SourcePath))
	}

	// Add document count if available
	if status != nil && status.DocumentCount > 0 {
		descParts = append(descParts, fmt.Sprintf("%d documents", status.DocumentCount))
	}

	if len(descParts) == 0 {
		return ""
	}

	return strings.Join(descParts, ", ")
}

// buildStoreDescriptionFromStoreSafe is a safe version that recovers from panics
func (t *SearchTool) buildStoreDescriptionFromStoreSafe(store *hectorcontext.DocumentStore, storeName string) string {
	defer func() {
		if r := recover(); r != nil {
			// Silently return empty description on panic
			_ = r // Acknowledge recovery value to satisfy linter
		}
	}()
	return t.buildStoreDescriptionFromStore(store, storeName)
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
			"source":    "local",
			"tool_type": "search",
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
