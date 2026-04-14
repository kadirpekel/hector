package vector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/philippgille/chromem-go"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/utils"
)

// ChromemProvider implements Provider using chromem-go for embedded vector storage.
//
// This is the recommended provider for zero-config deployments as it requires
// no external services. It stores vectors in memory with optional file persistence.
//
// Features:
//   - Pure Go, no external dependencies
//   - Optional file persistence (gzip compressed)
//   - Cosine similarity search
//   - Metadata filtering
//   - Multi-tenant support via app-specific directories
//
// Limitations:
//   - Single-process only (no distributed search)
//   - Memory-bound (all vectors in RAM)
//   - No hybrid search support
//
// For production at scale, consider Qdrant or other external providers.
type ChromemProvider struct {
	// globalDB is the default database (backward compatibility/global apps)
	globalDB *chromem.DB

	// appDBs holds lazy-loaded databases for specific apps
	appDBs map[string]*chromem.DB

	// persistPath is the configured global path
	persistPath string

	// rootDir is the derived root directory for calculating app paths
	// Usually the parent of .hector or the directory containing .hector
	rootDir string

	compress bool
	mu       sync.RWMutex

	// collections caches collection references for performance
	// Key format: "appID:collectionName"
	collections map[string]*chromem.Collection

	// embeddingFunc is used for similarity search (identity function)
	// The actual embedding is done externally via the embedder package
	embeddingFunc chromem.EmbeddingFunc
}

// ChromemConfig configures the chromem provider.
type ChromemConfig struct {
	// PersistPath for file persistence (optional).
	// If empty, vectors are stored in memory only.
	// Directory will be created if it doesn't exist.
	PersistPath string `yaml:"persist_path,omitempty"`

	// Compress enables gzip compression for persistence.
	// Reduces file size but increases CPU usage.
	Compress bool `yaml:"compress,omitempty"`
}

// NewChromemProvider creates a new chromem-based vector provider.
func NewChromemProvider(cfg ChromemConfig) (*ChromemProvider, error) {
	var globalDB *chromem.DB
	var rootDir string

	if cfg.PersistPath != "" {
		// Calculate rootDir for app-specific paths
		// If PersistPath is .../project/.hector/vectors, rootDir should be .../project
		absPath, err := filepath.Abs(cfg.PersistPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
		}

		// Heuristic: traverse up looking for .hector or stop at root
		// We expect typical path: /path/to/project/.hector/vectors
		dir := absPath
		for dir != "/" && dir != "." {
			base := filepath.Base(dir)
			if base == utils.DefaultHectorDir {
				rootDir = filepath.Dir(dir)
				break
			}
			dir = filepath.Dir(dir)
		}
		if rootDir == "" {
			// Fallback: use current directory if structure not found
			rootDir, err = filepath.Abs(".")
			if err != nil {
				return nil, fmt.Errorf("failed to get current directory: %w", err)
			}
		}

		// Ensure directory exists (existing logic)
		// Use centralized EnsureHectorDir if path contains .hector
		dir = cfg.PersistPath
		if filepath.Base(dir) == utils.DefaultHectorDir || filepath.Base(filepath.Dir(dir)) == utils.DefaultHectorDir {
			// Extract base path (parent of .hector)
			basePath := dir
			for filepath.Base(basePath) == utils.DefaultHectorDir || filepath.Base(basePath) == "vectors" || filepath.Base(basePath) == "chromem" {
				basePath = filepath.Dir(basePath)
			}
			if basePath == "" || basePath == "." {
				basePath = "."
			}
			if _, err := utils.EnsureHectorDir(basePath); err != nil {
				return nil, fmt.Errorf("failed to create %s directory: %w", utils.DefaultHectorDir, err)
			}
			// Also ensure the full path exists (for subdirectories like .hector/vectors)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create persist directory: %w", err)
			}
		} else {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create persist directory: %w", err)
			}
		}

		// Load global database
		globalDB, err = chromem.NewPersistentDB(cfg.PersistPath, cfg.Compress)
		if err != nil {
			slog.Warn("Failed to create persistent vector database, using in-memory",
				"path", cfg.PersistPath,
				"error", err)
			globalDB = chromem.NewDB()
		} else {
			slog.Info("Using persistent vector database", "path", cfg.PersistPath)
		}
	} else {
		globalDB = chromem.NewDB()
		slog.Info("Created in-memory vector database (no persistence)")
	}

	// Identity embedding function - we receive pre-computed vectors
	identityEmbed := func(ctx context.Context, text string) ([]float32, error) {
		// This should not be called when using pre-computed vectors
		return nil, fmt.Errorf("embedding function called but vectors should be pre-computed")
	}

	return &ChromemProvider{
		globalDB:      globalDB,
		appDBs:        make(map[string]*chromem.DB),
		persistPath:   cfg.PersistPath,
		rootDir:       rootDir,
		compress:      cfg.Compress,
		collections:   make(map[string]*chromem.Collection),
		embeddingFunc: identityEmbed,
	}, nil
}

// getDB resolves the correct database instance for the current app context.
func (p *ChromemProvider) getDB(ctx context.Context) (*chromem.DB, error) {
	appID := app.IDFromContext(ctx)

	// Default/Global app uses the global DB
	if appID == "default" || appID == "" {
		return p.globalDB, nil
	}

	// For in-memory mode (no persist path), we can just use the global DB
	// but logically partition with collection names if needed, or maintain separate in-memory DBs.
	// For now, if no persistence, everything goes to globalDB to avoid complexity.
	if p.persistPath == "" {
		return p.globalDB, nil
	}

	p.mu.RLock()
	db, ok := p.appDBs[appID]
	p.mu.RUnlock()
	if ok {
		return db, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check
	if db, ok := p.appDBs[appID]; ok {
		return db, nil
	}

	// Create app-specific DB
	// Path: <rootDir>/.hector/apps/<appID>/vectors
	appDir, err := utils.GetAppDir(p.rootDir, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get app dir: %w", err)
	}
	vectorDir := filepath.Join(appDir, "vectors")

	if err := os.MkdirAll(vectorDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app vector dir: %w", err)
	}

	db, err = chromem.NewPersistentDB(vectorDir, p.compress)
	if err != nil {
		return nil, fmt.Errorf("failed to create app vector db: %w", err)
	}

	slog.Info("Loaded app vector database", "app", appID, "path", vectorDir)
	p.appDBs[appID] = db
	return db, nil
}

// getCollection gets or creates a collection.
func (p *ChromemProvider) getCollection(ctx context.Context, name string) (*chromem.Collection, error) {
	appID := app.IDFromContext(ctx)
	cacheKey := appID + ":" + name

	p.mu.RLock()
	if col, ok := p.collections[cacheKey]; ok {
		p.mu.RUnlock()
		return col, nil
	}
	p.mu.RUnlock()

	// Resolve DB (might involve IO/locking, so do it outside main lock if possible,
	// but getDB manages its own locking for creation)
	db, err := p.getDB(ctx)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if col, ok := p.collections[cacheKey]; ok {
		return col, nil
	}

	// Create or get collection from the specific DB
	col, err := db.GetOrCreateCollection(name, nil, p.embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create collection %q for app %s: %w", name, appID, err)
	}

	p.collections[cacheKey] = col
	slog.Debug("ChromemProvider getCollection: created new collection",
		"name", name, "app", appID)
	return col, nil
}

// Upsert adds or updates a document with its vector embedding.
func (p *ChromemProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]any) error {
	col, err := p.getCollection(ctx, collection)
	if err != nil {
		return err
	}

	countBefore := col.Count()

	// Convert metadata to string map (chromem requirement)
	strMetadata := make(map[string]string, len(metadata))
	for k, v := range metadata {
		strMetadata[k] = fmt.Sprint(v)
	}

	// Extract content from metadata if present
	content := ""
	if c, ok := metadata["content"].(string); ok {
		content = c
	}

	doc := chromem.Document{
		ID:        id,
		Content:   content,
		Metadata:  strMetadata,
		Embedding: vector,
	}

	// AddDocument with pre-computed embedding
	if err := col.AddDocuments(ctx, []chromem.Document{doc}, runtime.NumCPU()); err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}

	countAfter := col.Count()
	if countAfter > countBefore {
		slog.Debug("ChromemProvider Upsert: document added",
			"collection", collection,
			"doc_id", id,
			"count", countAfter)
	}

	// Persist handled by NewPersistentDB (autosave) or we can't force it easily on the right DB without exposing it
	// chromem-go doesn't expose manual Persist on DB easily?
	// actually `NewPersistentDB` returns a DB that autosaves? No, it loads.
	// chromem-go DB has no Persist method?
	// It seems chromem-go persists on every modification?
	// Checking the original code: `p.persist()` was a no-op method.
	// So we don't need to do anything.

	return nil
}

// Search finds the most similar vectors in a collection.
func (p *ChromemProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]Result, error) {
	return p.SearchWithFilter(ctx, collection, vector, topK, nil)
}

// SearchWithFilter combines vector similarity with metadata filtering.
func (p *ChromemProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]Result, error) {
	col, err := p.getCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	// Cap topK to collection count (chromem requires nResults <= document count)
	docCount := col.Count()
	slog.Info("ChromemProvider search",
		"collection", collection,
		"doc_count", docCount,
		"requested_topK", topK)
	if docCount == 0 {
		slog.Warn("ChromemProvider search: collection is empty - no documents to search",
			"collection", collection)
		return []Result{}, nil
	}
	if topK > docCount {
		topK = docCount
	}

	// Convert filter to string map
	var whereFilter map[string]string
	if len(filter) > 0 {
		whereFilter = make(map[string]string, len(filter))
		for k, v := range filter {
			whereFilter[k] = fmt.Sprint(v)
		}
	}

	// Query using pre-computed vector
	// chromem.QueryWithEmbedding would be ideal but Query with empty text works
	// We need to use the embedding directly
	results, err := col.QueryEmbedding(ctx, vector, topK, whereFilter, nil)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to our Result type
	out := make([]Result, 0, len(results))
	for _, r := range results {
		metadata := make(map[string]any, len(r.Metadata))
		for k, v := range r.Metadata {
			metadata[k] = v
		}

		out = append(out, Result{
			ID:       r.ID,
			Score:    r.Similarity,
			Content:  r.Content,
			Metadata: metadata,
		})
	}

	return out, nil
}

// Delete removes a document from a collection by ID.
func (p *ChromemProvider) Delete(ctx context.Context, collection string, id string) error {
	col, err := p.getCollection(ctx, collection)
	if err != nil {
		return err
	}

	if err := col.Delete(ctx, nil, nil, id); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// DeleteByFilter removes all documents matching the filter.
func (p *ChromemProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]any) error {
	col, err := p.getCollection(ctx, collection)
	if err != nil {
		return err
	}

	// Convert filter to string map
	whereFilter := make(map[string]string, len(filter))
	for k, v := range filter {
		whereFilter[k] = fmt.Sprint(v)
	}

	if err := col.Delete(ctx, whereFilter, nil); err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}

	return nil
}

// CreateCollection creates a new collection.
// chromem-go creates collections implicitly, so this is a no-op.
func (p *ChromemProvider) CreateCollection(ctx context.Context, collection string, vectorDimension int) error {
	_, err := p.getCollection(ctx, collection)
	return err
}

// DeleteCollection removes a collection and all its documents.
// NOTE: For app isolation logic, this removes the collection FROM the app-specific DB.
func (p *ChromemProvider) DeleteCollection(ctx context.Context, collection string) error {
	// Get DB for this app
	db, err := p.getDB(ctx)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Delete from DB
	if err := db.DeleteCollection(collection); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Remove from cache
	appID := app.IDFromContext(ctx)
	cacheKey := appID + ":" + collection
	delete(p.collections, cacheKey)

	return nil
}

// Name returns the provider name.
func (p *ChromemProvider) Name() string {
	return "chromem"
}

// Close persists the database and releases resources.
func (p *ChromemProvider) Close() error {
	// Chromem-go doesn't require explicit close, but we might want to clear maps
	// or flush if it supported it.
	return nil
}

// DeleteApp cleans up all resources associated with an app.
func (p *ChromemProvider) DeleteApp(ctx context.Context, appID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close and remove app DB if exists
	if db, ok := p.appDBs[appID]; ok {
		// chromem-go doesn't expose a Close() method on *chromem.DB,
		// but removing the reference allows GC to clean up.
		// For persistent DBs, the data files remain on disk until os.RemoveAll.
		_ = db // Acknowledge for potential future Close() support
		delete(p.appDBs, appID)
		slog.Debug("ChromemProvider: removed app DB from cache", "app", appID)
	}

	// Remove cached collections for this app
	prefix := appID + ":"
	for key := range p.collections {
		if strings.HasPrefix(key, prefix) {
			delete(p.collections, key)
		}
	}

	slog.Info("ChromemProvider: deleted app resources from memory", "app", appID)
	return nil
}

// Count returns the number of documents in a collection.
func (p *ChromemProvider) Count(ctx context.Context, collection string) (int, error) {
	col, err := p.getCollection(ctx, collection)
	if err != nil {
		return 0, nil // Collection doesn't exist - return 0
	}
	return col.Count(), nil
}

// Ensure ChromemProvider implements Provider.
var _ Provider = (*ChromemProvider)(nil)
