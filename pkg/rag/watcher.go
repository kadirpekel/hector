// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches a directory for file changes using fsnotify.
//
// Direct port from legacy pkg/context/document_store.go fsnotify watching
type FileWatcher struct {
	watcher       *fsnotify.Watcher
	basePath      string
	filter        FileFilter
	eventChan     chan DocumentEvent
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isWatching    bool
	debounceDelay time.Duration
}

// FileWatcherConfig configures the file watcher.
type FileWatcherConfig struct {
	BasePath      string
	Filter        FileFilter
	DebounceDelay time.Duration // Delay before processing events (default: 100ms)
}

// NewFileWatcher creates a new file watcher.
func NewFileWatcher(cfg FileWatcherConfig) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	debounce := cfg.DebounceDelay
	if debounce == 0 {
		debounce = 100 * time.Millisecond
	}

	return &FileWatcher{
		watcher:       watcher,
		basePath:      cfg.BasePath,
		filter:        cfg.Filter,
		eventChan:     make(chan DocumentEvent, 100),
		debounceDelay: debounce,
	}, nil
}

// Start begins watching the directory for changes.
func (fw *FileWatcher) Start(ctx context.Context) (<-chan DocumentEvent, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.isWatching {
		return fw.eventChan, nil
	}

	fw.ctx, fw.cancel = context.WithCancel(ctx)
	fw.isWatching = true

	// Setup watching on all directories
	if err := fw.setupWatching(); err != nil {
		fw.isWatching = false
		return nil, err
	}

	// Start the event processing goroutine
	go fw.watchEvents()

	slog.Info("Started file watcher", "path", fw.basePath)

	return fw.eventChan, nil
}

// Stop stops watching for changes.
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isWatching {
		return nil
	}

	fw.cancel()
	fw.isWatching = false

	if err := fw.watcher.Close(); err != nil {
		return err
	}

	close(fw.eventChan)

	slog.Info("Stopped file watcher", "path", fw.basePath)

	return nil
}

// setupWatching adds the base path and all subdirectories to the watcher.
func (fw *FileWatcher) setupWatching() error {
	// Add the base path
	if err := fw.watcher.Add(fw.basePath); err != nil {
		return err
	}

	// Walk and add all subdirectories
	return filepath.Walk(fw.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if should be excluded
		if fw.filter != nil && fw.filter.ShouldExclude(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add directories to watcher
		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				slog.Warn("Failed to watch directory", "path", path, "error", err)
			}
		}

		return nil
	})
}

// watchEvents processes fsnotify events.
func (fw *FileWatcher) watchEvents() {
	// Debounce map to coalesce rapid events
	pendingEvents := make(map[string]fsnotify.Event)
	var pendingMu sync.Mutex
	var debounceTimer *time.Timer

	processEvents := func() {
		pendingMu.Lock()
		events := pendingEvents
		pendingEvents = make(map[string]fsnotify.Event)
		pendingMu.Unlock()

		for _, event := range events {
			fw.handleFileEvent(event)
		}
	}

	for {
		select {
		case <-fw.ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			// Process any remaining events
			processEvents()
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Skip chmod events
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			// Add to pending events
			pendingMu.Lock()
			pendingEvents[event.Name] = event
			pendingMu.Unlock()

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(fw.debounceDelay, processEvents)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "path", fw.basePath, "error", err)
			fw.eventChan <- DocumentEvent{
				Type:  DocumentEventError,
				Error: err,
			}
		}
	}
}

// handleFileEvent processes a single fsnotify event.
func (fw *FileWatcher) handleFileEvent(event fsnotify.Event) {
	path := event.Name

	// Check filters
	if fw.filter != nil {
		if fw.filter.ShouldExclude(path) {
			return
		}
		if !fw.filter.ShouldInclude(path) {
			return
		}
	}

	var eventType DocumentEventType
	var doc Document

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = DocumentEventCreate

		// Check if it's a directory and add to watcher
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				slog.Warn("Failed to watch new directory", "path", path, "error", err)
			}
			return // Don't emit event for directories
		}

		// Read the file
		doc = fw.readDocument(path)

	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = DocumentEventUpdate
		doc = fw.readDocument(path)

	case event.Op&fsnotify.Remove == fsnotify.Remove, event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = DocumentEventDelete
		relPath, _ := filepath.Rel(fw.basePath, path)
		doc = Document{
			ID:         path,
			SourcePath: relPath,
		}

	default:
		return
	}

	select {
	case fw.eventChan <- DocumentEvent{
		Type:     eventType,
		Document: doc,
	}:
	case <-fw.ctx.Done():
		return
	default:
		slog.Warn("Event channel full, dropping event", "path", path, "event", eventType)
	}
}

// readDocument reads a file and creates a Document.
func (fw *FileWatcher) readDocument(path string) Document {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("Failed to stat file", "path", path, "error", err)
		return Document{ID: path}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("Failed to read file", "path", path, "error", err)
		return Document{ID: path}
	}

	relPath, _ := filepath.Rel(fw.basePath, path)

	return Document{
		ID:         path,
		Content:    string(content),
		SourcePath: relPath,
		MimeType:   detectMimeType(path),
		Size:       info.Size(),
		Metadata: map[string]any{
			"path":          path,
			"rel_path":      relPath,
			"name":          info.Name(),
			"absolute_path": path,
			"last_modified": info.ModTime().Unix(),
		},
	}
}

// IsWatching returns whether the watcher is active.
func (fw *FileWatcher) IsWatching() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.isWatching
}
