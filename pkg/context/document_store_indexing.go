package context

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/kadirpekel/hector/pkg/context/indexing"
)

// indexFromDataSource indexes documents from any data source (directory, SQL, API)
func (ds *DocumentStore) indexFromDataSource() error {

	// Load existing index state for incremental indexing (directory sources only)
	// Load this early so we can use it when evaluating checkpoint completion
	var existingDocs map[string]FileIndexInfo
	ctx := ds.ctx
	var err error
	if ds.dataSource.Type() == "directory" {
		existingDocs, err = ds.loadIndexState()
		if err != nil {
			slog.Warn("Failed to load index state, performing full reindex", "error", err)
			existingDocs = make(map[string]FileIndexInfo)
		}
	} else {
		existingDocs = make(map[string]FileIndexInfo)
	}

	useIncrementalIndexing := ds.config.EnableIncrementalIndexing != nil && *ds.config.EnableIncrementalIndexing && ds.dataSource.SupportsIncrementalIndexing()

	// Try to load checkpoint (only for directory sources with file tracking)
	var checkpoint *IndexCheckpoint
	if ds.dataSource.Type() == "directory" {
		checkpoint, err = ds.checkpointManager.LoadCheckpoint()
		if checkpoint != nil && err == nil {
			processedCount := len(checkpoint.ProcessedFiles)
			// Only clear checkpoint if it's truly complete (all files processed)
			// Completed checkpoints are cleared because index state (saved after successful indexing)
			// serves as the persistent record for subsequent runs. Checkpoints are only for
			// resuming interrupted indexing runs, not for skipping files on subsequent runs.
			if processedCount > 0 && checkpoint.TotalFiles > 0 && processedCount >= checkpoint.TotalFiles {
				// Checkpoint is complete - previous indexing finished successfully
				// Clear it since index state should be available for incremental indexing
				// If index state is missing, incremental indexing will handle it gracefully
				_ = ds.checkpointManager.ClearCheckpoint()
				checkpoint = nil
			} else {
				// Incomplete checkpoint - resume from where we left off
				slog.Info("Resuming from checkpoint", "info", ds.checkpointManager.FormatCheckpointInfo(checkpoint))
			}
		}
	}

	if len(existingDocs) > 0 && useIncrementalIndexing {
		slog.Info("Incremental indexing enabled", "existing_documents", len(existingDocs))
	} else if len(existingDocs) > 0 {
		// existingDocs > 0 but useIncrementalIndexing is false
		slog.Info("Found existing documents but incremental indexing is disabled", "existing_documents", len(existingDocs))
	} else {
		slog.Info("First indexing or full reindex mode")
	}

	// Discover documents from data source
	docChan, errChan := ds.dataSource.DiscoverDocuments(ctx)

	// Phase 1: Collect all documents first (discovery phase)
	// This ensures TotalFiles is set before processing starts for accurate progress tracking
	slog.Info("Discovering documents...")
	var allDocs []indexing.Document

	// Drain error channel in background
	go func() {
		for err := range errChan {
			// Discovery errors: files that couldn't be read/discovered
			slog.Debug("Discovery error", "error", err)
		}
	}()

	for doc := range docChan {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		allDocs = append(allDocs, doc)
	}

	// Set total files now that discovery is complete
	totalFiles := int64(len(allDocs))
	ds.progressTracker.SetTotalFiles(totalFiles)
	if ds.dataSource.Type() == "directory" {
		ds.checkpointManager.SetTotalFiles(int(totalFiles))
	}

	slog.Info("Discovery complete", "total_documents", totalFiles)

	// Phase 2: Process all discovered documents
	ds.progressTracker.Start()
	defer func() {
		ds.progressTracker.Stop()
		if ds.dataSource.Type() == "directory" {
			_ = ds.checkpointManager.SaveCheckpoint()
		}
	}()

	var indexedCount sync.WaitGroup
	foundDocs := make(map[string]bool)
	indexedDocs := make(map[string]FileIndexInfo)
	var docsMu sync.Mutex

	failedDocs := make([]string, 0)
	var failedDocsMu sync.Mutex

	for _, doc := range allDocs {
		select {
		case <-ctx.Done():
			indexedCount.Wait()
			return ctx.Err()
		default:
		}

		if !doc.ShouldIndex {
			ds.progressTracker.IncrementSkipped()
			ds.progressTracker.IncrementProcessed()
			continue
		}

		// Check incremental indexing for directory sources
		if ds.dataSource.Type() == "directory" && useIncrementalIndexing {
			relPath := ds.getRelPath(&doc)
			if relPath != "" {
				if docInfo, exists := existingDocs[relPath]; exists {
					// Compare Unix timestamps (seconds) to avoid precision issues
					// File mod times are stored as Unix timestamps in index state
					currentModTime := doc.LastModified.Unix()
					storedModTime := docInfo.ModTime
					// Also check size to be sure (fast check)
					if currentModTime <= storedModTime && doc.Size == docInfo.Size {
						// File hasn't changed, but check if extractors are still available
						// If extractors are not available (e.g., MCP server is down), don't skip
						// so that indexing errors are visible to the user
						mimeType := ds.detectMIMEType(doc.ID)
						if ds.contentExtractors.HasExtractorForFile(doc.ID, mimeType) {
							// File hasn't changed and extractors are available - skip it but mark it as found
							// so it's preserved in the index state
							// Also record it in checkpoint so checkpoint stays in sync
							docsMu.Lock()
							foundDocs[relPath] = true
							foundDocs[doc.ID] = true
							docsMu.Unlock()
							ds.progressTracker.IncrementSkipped()
							ds.progressTracker.IncrementProcessed()
							// Record skipped file in checkpoint to keep it in sync
							ds.checkpointManager.RecordFile(relPath, doc.Size, doc.LastModified, "skipped")
							_ = ds.checkpointManager.SaveCheckpoint()
							continue
						}
						// Extractors not available - don't skip, let indexing attempt so errors are visible
					}
				}
			}
		}

		// Check checkpoint for directory sources
		if ds.dataSource.Type() == "directory" && checkpoint != nil {
			relPath := ds.getRelPath(&doc)
			if relPath != "" && !ds.checkpointManager.ShouldProcessFile(relPath, doc.Size, doc.LastModified) {
				// File was already processed in checkpoint and hasn't changed
				// Count it as processed and skipped (it's already done) and mark it as found
				docsMu.Lock()
				foundDocs[relPath] = true
				foundDocs[doc.ID] = true
				docsMu.Unlock()
				ds.progressTracker.IncrementSkipped()
				ds.progressTracker.IncrementProcessed()
				continue
			}
			// File is in checkpoint but was modified - we'll process it and increment processed
			// This is correct because we'll replace the old checkpoint entry
		}

		docsMu.Lock()
		// Use relative path for foundDocs to match existingDocs keys
		relPath := ds.getRelPath(&doc)
		if relPath != "" {
			foundDocs[relPath] = true
		}
		// Also store by absolute path for backward compatibility
		foundDocs[doc.ID] = true
		docsMu.Unlock()

		ds.indexingSemaphore <- struct{}{}
		indexedCount.Add(1)

		go func(d indexing.Document) {
			defer func() {
				if r := recover(); r != nil {
					// Don't log panic during indexing to avoid breaking progress bar display
					// Panic info will be available in failedDocs list
					ds.progressTracker.IncrementFailed()
					ds.progressTracker.IncrementProcessed()
					failedDocsMu.Lock()
					failedDocs = append(failedDocs, d.ID)
					failedDocsMu.Unlock()
				}
				<-ds.indexingSemaphore
				indexedCount.Done()
			}()

			// Update current file in progress tracker
			if ds.dataSource.Type() == "directory" {
				relPath := ds.getRelPath(&d)
				ds.progressTracker.SetCurrentFile(relPath)
			}

			// Index document and track extractor usage
			extractorName, err := ds.indexDocument(&d)
			if err != nil {
				ds.progressTracker.IncrementFailed()
				ds.progressTracker.IncrementProcessed()
				failedDocsMu.Lock()
				failedDocs = append(failedDocs, fmt.Sprintf("%s: %v", d.ID, err))
				failedDocsMu.Unlock()

				// Log error details for debugging (at debug level to avoid breaking progress bar)
				slog.Debug("Document indexing failed",
					"document_id", d.ID,
					"error", err.Error(),
					"extractor", extractorName)
				// Errors will be shown in the final summary
			} else {
				ds.progressTracker.IncrementIndexed()
				ds.progressTracker.IncrementProcessed()
				// Track extractor usage for statistics
				ds.progressTracker.RecordExtractorUsage(extractorName)

				if ds.dataSource.Type() == "directory" {
					relPath := ds.getRelPath(&d)
					if relPath != "" {
						docsMu.Lock()
						indexedDocs[relPath] = FileIndexInfo{
							ModTime: d.LastModified.Unix(),
							Size:    d.Size,
							Hash:    ds.computeDocumentHash(&d),
						}
						docsMu.Unlock()
						ds.checkpointManager.RecordFile(relPath, d.Size, d.LastModified, "indexed")
						_ = ds.checkpointManager.SaveCheckpoint()
					}
				}
			}
		}(doc)
	}

	// Wait for all indexing to complete
	indexedCount.Wait()

	// Cleanup deleted documents (directory sources only)
	if ds.dataSource.Type() == "directory" {
		deletedDocs, cleanedUpDocs, err := ds.cleanupDeletedFiles(ctx, existingDocs, foundDocs)
		if err != nil {
			slog.Warn("Cleanup of deleted files failed", "error", err)
		}
		if len(deletedDocs) > 0 {
			slog.Info("Cleaned up deleted documents from index", "count", len(deletedDocs), "index", ds.name)
		}

		// Save index state
		finalState := make(map[string]FileIndexInfo)
		for path, info := range existingDocs {
			if foundDocs[path] {
				finalState[path] = info
			} else if !cleanedUpDocs[path] {
				finalState[path] = info
			}
		}
		for path, info := range indexedDocs {
			finalState[path] = info
		}
		indexStateSaved := false
		if err := ds.saveIndexState(finalState, len(finalState)*3); err != nil {
			slog.Warn("Failed to save index state", "error", err)
		} else {
			indexStateSaved = true
		}

		// Clear checkpoint after successful indexing if index state was saved
		// Only clear if checkpoint is complete (all files processed) or if incremental indexing
		// is enabled and working (index state contains all files)
		// This ensures we don't lose checkpoint information for incomplete runs
		if indexStateSaved {
			shouldClear := false
			if checkpoint != nil {
				processedCount := len(checkpoint.ProcessedFiles)
				// Clear if checkpoint is complete OR if incremental indexing is enabled
				// (in which case index state is the source of truth)
				if processedCount > 0 && checkpoint.TotalFiles > 0 && processedCount >= checkpoint.TotalFiles {
					shouldClear = true // Checkpoint is complete
				} else if useIncrementalIndexing && len(finalState) > 0 {
					shouldClear = true // Incremental indexing working, index state is source of truth
				}
				// Otherwise keep checkpoint for resume capability
			} else if useIncrementalIndexing && len(finalState) > 0 {
				// Checkpoint was cleared earlier, but ensure any remaining file is cleared
				shouldClear = true
			}
			if shouldClear {
				_ = ds.checkpointManager.ClearCheckpoint()
			}
		}
	}

	// Update status
	stats := ds.progressTracker.GetStats()
	ds.mu.Lock()
	ds.status.DocumentCount = int(stats.IndexedFiles)
	ds.mu.Unlock()

	// Print summary
	if len(failedDocs) > 0 {
		maxShow := 10
		slog.Warn("Failed Documents", "count", len(failedDocs))
		for i, docID := range failedDocs {
			if i >= maxShow {
				remaining := len(failedDocs) - maxShow
				fmt.Printf("   ... and %d more documents (check logs for details)\n", remaining)
				break
			}
			slog.Error("Failed to index document", "document_id", docID)
		}
	}

	return nil
}

// getRelPath extracts the relative path from a document, with fallbacks
func (ds *DocumentStore) getRelPath(doc *indexing.Document) string {
	if doc.SourcePath != "" {
		return doc.SourcePath
	}
	if pathVal, ok := doc.Metadata["rel_path"].(string); ok {
		return pathVal
	}
	if ds.dataSource.Type() == "directory" {
		if rel, err := filepath.Rel(ds.sourcePath, doc.ID); err == nil {
			return rel
		}
	}
	return ""
}

// computeDocumentHash computes a hash for a document (for change detection)
func (ds *DocumentStore) computeDocumentHash(doc *indexing.Document) string {
	// For directory sources, use file path; for others, use content hash
	if ds.dataSource.Type() == "directory" {
		return ds.computeFileHash(doc.ID)
	}
	// For SQL/API sources, hash the content
	hash := md5.Sum([]byte(doc.Content))
	return fmt.Sprintf("%x", hash)
}
