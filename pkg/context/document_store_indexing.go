package context

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/context/indexing"
)

// indexFromDataSource indexes documents from any data source (directory, SQL, API)
func (ds *DocumentStore) indexFromDataSource() error {

	// Try to load checkpoint (only for directory sources with file tracking)
	var checkpoint *IndexCheckpoint
	ctx := ds.ctx
	var err error
	if ds.dataSource.Type() == "directory" {
		checkpoint, err = ds.checkpointManager.LoadCheckpoint()
		if checkpoint != nil && err == nil {
			processedCount := len(checkpoint.ProcessedFiles)
			// Only clear checkpoint if it's truly complete (all files processed)
			// Don't clear based on TotalFiles comparison as it might be inaccurate
			if processedCount > 0 && checkpoint.TotalFiles > 0 && processedCount >= checkpoint.TotalFiles {
				// Verify by checking if we actually have more files to process
				// This prevents clearing checkpoint prematurely
				_ = ds.checkpointManager.ClearCheckpoint()
				checkpoint = nil
			} else {
				fmt.Println("🔄 " + ds.checkpointManager.FormatCheckpointInfo(checkpoint))
				fmt.Println("   Resuming from checkpoint...")
			}
		}
	}

	// Load existing index state for incremental indexing (directory sources only)
	var existingDocs map[string]FileIndexInfo
	if ds.dataSource.Type() == "directory" {
		existingDocs, err = ds.loadIndexState()
		if err != nil {
			log.Printf("Warning: Failed to load index state, performing full reindex: %v", err)
			existingDocs = make(map[string]FileIndexInfo)
		}
	} else {
		existingDocs = make(map[string]FileIndexInfo)
	}

	useIncrementalIndexing := ds.config.IncrementalIndexing != nil && *ds.config.IncrementalIndexing && ds.dataSource.SupportsIncrementalIndexing()

	if len(existingDocs) > 0 && useIncrementalIndexing {
		fmt.Printf("📊 Incremental indexing: Found %d existing document(s) in index\n", len(existingDocs))
	} else if len(existingDocs) > 0 {
		fmt.Printf("📊 Found %d existing document(s) in index (will be reindexed)\n", len(existingDocs))
	} else {
		fmt.Printf("📊 First indexing or full reindex mode\n")
	}

	// Discover documents from data source
	docChan, errChan := ds.dataSource.DiscoverDocuments(ctx)

	// Count total documents first (for progress tracking)
	// For directory sources, we can count files; for others, we'll estimate
	var totalDocs int64
	if ds.dataSource.Type() == "directory" {
		// When resuming from checkpoint OR using incremental indexing, count ALL files
		// that will be discovered (including unchanged files that will be skipped).
		// This ensures the progress percentage is accurate.
		// Otherwise, count only files that need processing.
		if checkpoint != nil || useIncrementalIndexing {
			totalDocs = ds.countAllDirectoryFiles(ctx)
		} else {
			totalDocs = ds.countDirectoryFiles(ctx, existingDocs, checkpoint, useIncrementalIndexing)
		}
	} else {
		// For SQL/API, we'll count as we process
		totalDocs = 0
	}

	// Set up progress tracking
	// When using checkpoint or incremental indexing, totalDocs already includes all files
	ds.progressTracker.SetTotalFiles(totalDocs)
	if ds.dataSource.Type() == "directory" {
		ds.checkpointManager.SetTotalFiles(int(totalDocs))
	}

	ds.progressTracker.Start()
	defer func() {
		ds.progressTracker.Stop()
		if ds.dataSource.Type() == "directory" {
			_ = ds.checkpointManager.SaveCheckpoint()
		}
	}()

	// Process documents
	var indexedCount sync.WaitGroup
	foundDocs := make(map[string]bool)
	indexedDocs := make(map[string]FileIndexInfo)
	var docsMu sync.Mutex

	failedDocs := make([]string, 0)
	var failedDocsMu sync.Mutex

	// Track document count for non-directory sources
	docCount := int64(0)
	var docCountMu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case doc, ok := <-docChan:
			if !ok {
				// Channel closed, wait for all indexing to complete
				indexedCount.Wait()

				// Cleanup deleted documents (directory sources only)
				if ds.dataSource.Type() == "directory" {
					deletedDocs, cleanedUpDocs, err := ds.cleanupDeletedFiles(ctx, existingDocs, foundDocs)
					if err != nil {
						log.Printf("Warning: Cleanup of deleted files failed: %v", err)
					}
					if len(deletedDocs) > 0 {
						fmt.Printf("🗑️  Cleaned up %d deleted document(s) from index '%s'\n", len(deletedDocs), ds.name)
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
					if err := ds.saveIndexState(finalState, len(finalState)*3); err != nil {
						log.Printf("Warning: Failed to save index state: %v", err)
					}

					_ = ds.checkpointManager.ClearCheckpoint()
				}

				// Update status
				stats := ds.progressTracker.GetStats()
				ds.mu.Lock()
				ds.status.DocumentCount = int(stats.IndexedFiles)
				ds.mu.Unlock()

				// Print summary
				if len(failedDocs) > 0 {
					maxShow := 10
					fmt.Println("\n⚠️  Failed Documents:")
					for i, docID := range failedDocs {
						if i >= maxShow {
							remaining := len(failedDocs) - maxShow
							fmt.Printf("   ... and %d more documents (check logs for details)\n", remaining)
							break
						}
						fmt.Printf("   ❌ %s\n", docID)
					}
				}

				return nil
			}

			if !doc.ShouldIndex {
				ds.progressTracker.IncrementSkipped()
				continue
			}

			// Check incremental indexing for directory sources
			if ds.dataSource.Type() == "directory" && useIncrementalIndexing {
				// Use relative path for consistency with existingDocs
				relPath := doc.SourcePath
				if relPath == "" {
					if pathVal, ok := doc.Metadata["rel_path"].(string); ok {
						relPath = pathVal
					} else {
						// Fallback: compute relative path from absolute path
						if rel, err := filepath.Rel(ds.sourcePath, doc.ID); err == nil {
							relPath = rel
						}
					}
				}
				if relPath != "" {
					if docInfo, exists := existingDocs[relPath]; exists {
						if doc.LastModified.Before(time.Unix(docInfo.ModTime, 0)) || doc.LastModified.Equal(time.Unix(docInfo.ModTime, 0)) {
							// File hasn't changed - skip it but mark it as found
							// so it's preserved in the index state
							docsMu.Lock()
							foundDocs[relPath] = true
							foundDocs[doc.ID] = true
							docsMu.Unlock()
							ds.progressTracker.IncrementSkipped()
							continue
						}
					}
				}
			}

			// Check checkpoint for directory sources
			if ds.dataSource.Type() == "directory" && checkpoint != nil {
				relPath := doc.SourcePath
				if relPath == "" {
					if pathVal, ok := doc.Metadata["rel_path"].(string); ok {
						relPath = pathVal
					} else {
						// Fallback: compute relative path from absolute path
						if rel, err := filepath.Rel(ds.sourcePath, doc.ID); err == nil {
							relPath = rel
						}
					}
				}
				if relPath != "" && !ds.checkpointManager.ShouldProcessFile(relPath, doc.Size, doc.LastModified) {
					// File was already processed in checkpoint and hasn't changed
					// Count it as processed (it's already done) and mark it as found
					docsMu.Lock()
					foundDocs[relPath] = true
					foundDocs[doc.ID] = true
					docsMu.Unlock()
					ds.progressTracker.IncrementProcessed()
					continue
				}
				// File is in checkpoint but was modified - we'll process it and increment processed
				// This is correct because we'll replace the old checkpoint entry
			}

			// Update document count for non-directory sources
			if ds.dataSource.Type() != "directory" {
				docCountMu.Lock()
				docCount++
				if totalDocs == 0 {
					ds.progressTracker.SetTotalFiles(docCount)
				}
				docCountMu.Unlock()
			}

			docsMu.Lock()
			// Use relative path for foundDocs to match existingDocs keys
			relPath := doc.SourcePath
			if relPath == "" {
				if pathVal, ok := doc.Metadata["rel_path"].(string); ok {
					relPath = pathVal
				} else if ds.dataSource.Type() == "directory" {
					// Fallback: compute relative path from absolute path
					if rel, err := filepath.Rel(ds.sourcePath, doc.ID); err == nil {
						relPath = rel
					}
				}
			}
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
					relPath := d.SourcePath
					if relPath == "" {
						if pathVal, ok := d.Metadata["rel_path"].(string); ok {
							relPath = pathVal
						}
					}
					ds.progressTracker.SetCurrentFile(relPath)
				}

				if err := ds.indexDocument(&d); err != nil {
					ds.progressTracker.IncrementFailed()
					ds.progressTracker.IncrementProcessed()
					failedDocsMu.Lock()
					failedDocs = append(failedDocs, fmt.Sprintf("%s: %v", d.ID, err))
					failedDocsMu.Unlock()

					// Don't log errors during indexing to avoid breaking progress bar display
					// Errors will be shown in the final summary
				} else {
					ds.progressTracker.IncrementIndexed()
					ds.progressTracker.IncrementProcessed()

					if ds.dataSource.Type() == "directory" {
						relPath := d.SourcePath
						if relPath == "" {
							if pathVal, ok := d.Metadata["rel_path"].(string); ok {
								relPath = pathVal
							}
						}
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
			}(doc)

		case err, ok := <-errChan:
			if !ok {
				continue
			}
			ds.progressTracker.IncrementFailed()
			// Don't log discovery errors during indexing to avoid breaking progress bar display
			// These are typically non-fatal and don't need immediate attention
			_ = err // Error is tracked via IncrementFailed()
		}
	}
}

// countAllDirectoryFiles counts ALL files that will be discovered (for progress tracking)
// This includes files that will be skipped due to checkpoint or incremental indexing
// It uses the same filters as DirectorySource.DiscoverDocuments to ensure accurate counting
func (ds *DocumentStore) countAllDirectoryFiles(ctx context.Context) int64 {
	var count int64

	// Note: We don't filter by maxFileSize here because we want to count all files
	// that will be discovered, even if some will be filtered out later by maxFileSize.
	// Files that exceed maxFileSize will be skipped during discovery and won't increment
	// processed, so the total should include them for accurate progress tracking.

	err := filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Size() == 0 {
			return nil
		}
		// Apply same filters as DiscoverDocuments
		if ds.shouldExclude(path) || !ds.shouldInclude(path) {
			return nil
		}
		// Count all files that match filters - they will all be discovered
		// Some will be skipped due to checkpoint/incremental indexing/maxFileSize, but they still count toward total
		count++
		return nil
	})
	if err != nil {
		return 0
	}
	return count
}

// countDirectoryFiles counts files for directory sources (for progress tracking)
func (ds *DocumentStore) countDirectoryFiles(ctx context.Context, existingDocs map[string]FileIndexInfo, checkpoint *IndexCheckpoint, useIncremental bool) int64 {
	var count int64
	err := filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Size() == 0 {
			return nil
		}
		if ds.shouldExclude(path) || !ds.shouldInclude(path) {
			return nil
		}

		relPath, _ := filepath.Rel(ds.sourcePath, path)
		if checkpoint != nil && !ds.checkpointManager.ShouldProcessFile(relPath, info.Size(), info.ModTime()) {
			return nil
		}

		if useIncremental && !ds.shouldReindexFile(path, info.ModTime(), existingDocs) {
			return nil
		}

		count++
		return nil
	})
	if err != nil {
		return 0
	}
	return count
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
