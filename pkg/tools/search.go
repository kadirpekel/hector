package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

type SearchTool struct {
	config          *config.SearchToolConfig
	availableStores []string
}

type SearchRequest struct {
	Query       string            `json:"query"`
	Type        string            `json:"type"`
	Stores      []string          `json:"stores"`      // Document stores to search
	Collections []string          `json:"collections"` // Collection names to search directly
	Language    string            `json:"language"`
	Limit       int               `json:"limit"`
	Context     map[string]string `json:"context"`
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

// storeSearchResult holds results from a single store search
type storeSearchResult struct {
	storeName string
	results   []DocumentSearchResult
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
	var collectionsUsed []string

	// If collections are specified, search them directly in parallel
	if len(req.Collections) > 0 {
		// Search each collection directly in parallel
		// Try to find a document store that points to this collection to get the correct database
		availableStores := t.getAvailableStores()

		var wg sync.WaitGroup
		resultsChan := make(chan collectionSearchResult, len(req.Collections))

		// Launch parallel collection searches
		for _, collectionName := range req.Collections {
			wg.Add(1)
			go func(collName string) {
				defer wg.Done()

				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				var searchEngine *hectorcontext.SearchEngine
				var storeName string

				// Try to find a document store that points to this collection
				foundStore := false
				for name, store := range availableStores {
					if store.GetConfig().Collection == collName {
						searchEngine = store.GetSearchEngine()
						storeName = name
						foundStore = true
						break
					}
				}

				// If no store found for this collection, use first available store's search engine
				if !foundStore && len(availableStores) > 0 {
					for name, store := range availableStores {
						searchEngine = store.GetSearchEngine()
						if searchEngine != nil {
							storeName = name
							break
						}
					}
				}

				if searchEngine == nil {
					fmt.Printf("Warning: No search engine available for collection %s, skipping\n", collName)
					return
				}

				// Check for context cancellation before performing search
				select {
				case <-ctx.Done():
					return
				default:
				}

				results, err := t.searchCollection(ctx, searchEngine, collName, storeName, req)
				if err != nil {
					fmt.Printf("Error: Search in collection %s failed: %v\n", collName, err)
					return
				}

				// Check for context cancellation before sending results
				select {
				case <-ctx.Done():
					return
				case resultsChan <- collectionSearchResult{
					collectionName: collName,
					results:        results,
				}:
				}
			}(collectionName)
		}

		// Wait for all searches to complete and close channel
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results from all collections (channel collection is thread-safe)
		// Also handle context cancellation during result collection
		for {
			select {
			case <-ctx.Done():
				// Context cancelled, return partial results
				return t.buildSearchResponse(allResults, storesUsed, collectionsUsed, req, start)
			case result, ok := <-resultsChan:
				if !ok {
					// Channel closed, all results collected
					goto doneCollections
				}
				allResults = append(allResults, result.results...)
				collectionsUsed = append(collectionsUsed, result.collectionName)
			}
		}
	doneCollections:
	} else {
		// Search document stores in parallel
		availableStores := t.getAvailableStores()
		if len(availableStores) == 0 {
			return t.createErrorResponse("No document stores available")
		}

		storesToSearch := t.getStoresToSearch(req.Stores, availableStores)
		if len(storesToSearch) == 0 {
			return t.createErrorResponse("No matching document stores found")
		}

		// Use goroutines to search all stores in parallel
		var wg sync.WaitGroup
		resultsChan := make(chan storeSearchResult, len(storesToSearch))

		// Launch parallel searches
		for _, storeName := range storesToSearch {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()

				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				store, exists := hectorcontext.GetDocumentStoreFromRegistry(name)
				if !exists {
					fmt.Printf("Warning: Store %s not found in registry\n", name)
					return
				}

				// Check for context cancellation before performing search
				select {
				case <-ctx.Done():
					return
				default:
				}

				results, err := t.searchInStore(ctx, store, name, req)
				if err != nil {
					fmt.Printf("Error: Search in store %s failed: %v\n", name, err)
					return
				}

				// Check for context cancellation before sending results
				select {
				case <-ctx.Done():
					return
				case resultsChan <- storeSearchResult{
					storeName: name,
					results:   results,
				}:
				}
			}(storeName)
		}

		// Wait for all searches to complete and close channel
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Collect results from all stores (channel collection is thread-safe)
		// Also handle context cancellation during result collection
		for {
			select {
			case <-ctx.Done():
				// Context cancelled, return partial results
				return t.buildSearchResponse(allResults, storesUsed, collectionsUsed, req, start)
			case result, ok := <-resultsChan:
				if !ok {
					// Channel closed, all results collected
					goto doneStores
				}
				allResults = append(allResults, result.results...)
				storesUsed = append(storesUsed, result.storeName)
			}
		}
	doneStores:
	}

	return t.buildSearchResponse(allResults, storesUsed, collectionsUsed, req, start)
}

// buildSearchResponse builds the final search response with sorting and limiting
func (t *SearchTool) buildSearchResponse(allResults []DocumentSearchResult, storesUsed, collectionsUsed []string, req SearchRequest, start time.Time) (string, error) {
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

	// Add collections used to stores used for display
	if len(collectionsUsed) > 0 {
		response.StoresUsed = append(response.StoresUsed, collectionsUsed...)
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

		// Build store description
		storeDescription := t.buildStoreDescription(storeName)

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
			Metadata:         make(map[string]string),
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

// searchCollection searches a collection directly using the search engine
// storeName is the name of the document store that owns this collection (if known)
func (t *SearchTool) searchCollection(ctx context.Context, searchEngine *hectorcontext.SearchEngine, collectionName, storeName string, req SearchRequest) ([]DocumentSearchResult, error) {
	// Search the collection directly
	filter := map[string]interface{}{
		"collection": collectionName,
	}

	searchResults, err := searchEngine.SearchWithFilter(ctx, req.Query, req.Limit, filter)
	if err != nil {
		return nil, fmt.Errorf("collection search failed: %w", err)
	}

	// Convert search results to document search results
	var results []DocumentSearchResult
	for _, result := range searchResults {
		language := ""
		filePath := ""
		title := ""
		docType := ""
		resultStoreName := storeName // Default to provided store name

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
			// Prefer store_name from metadata (most accurate), fallback to provided storeName, then collection name
			if sn, ok := result.Metadata["store_name"].(string); ok && sn != "" {
				resultStoreName = sn
			} else if resultStoreName == "" {
				resultStoreName = collectionName // Fallback to collection name if no store name available
			}
		} else if resultStoreName == "" {
			resultStoreName = collectionName // Fallback to collection name if no metadata
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
				content = c
			}
		}

		if title == "" {
			if filePath != "" {
				title = filePath
			} else {
				title = result.ID
			}
		}

		matchType := req.Type
		if matchType == "" {
			matchType = "content"
		}

		metadata := make(map[string]string)
		if result.Metadata != nil {
			for k, v := range result.Metadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
		}

		// Ensure collection name is in metadata
		metadata["collection"] = collectionName

		// Build store description
		storeDescription := t.buildStoreDescription(resultStoreName)

		documentResult := DocumentSearchResult{
			DocumentID:       result.ID,
			StoreName:        resultStoreName,
			StoreDescription: storeDescription,
			FilePath:         filePath,
			Title:            title,
			Content:          content,
			Type:             docType,
			Language:         language,
			Score:            float64(result.Score),
			StartLine:        startLine,
			EndLine:          endLine,
			MatchType:        matchType,
			Metadata:         metadata,
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
				Name:        "collections",
				Type:        "array",
				Description: "Collection names to search directly (bypasses document stores)",
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

// buildStoreDescription builds a description for a document store from its config and status
func (t *SearchTool) buildStoreDescription(storeName string) string {
	if storeName == "" || storeName == "unknown" {
		return ""
	}

	// Try to get store from registry
	store, exists := hectorcontext.GetDocumentStoreFromRegistry(storeName)
	if !exists {
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

	if collections, ok := args["collections"]; ok {
		switch v := collections.(type) {
		case []interface{}:
			for _, collection := range v {
				if c, ok := collection.(string); ok {
					req.Collections = append(req.Collections, c)
				}
			}
		case []string:
			req.Collections = v
		case string:
			if v != "" {
				req.Collections = []string{v}
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
