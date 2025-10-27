package context

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexCheckpoint represents a saved indexing checkpoint
type IndexCheckpoint struct {
	Version        string                    `json:"version"`
	StoreName      string                    `json:"store_name"`
	SourcePath     string                    `json:"source_path"`
	StartTime      time.Time                 `json:"start_time"`
	LastUpdate     time.Time                 `json:"last_update"`
	ProcessedFiles map[string]FileCheckpoint `json:"processed_files"`
	TotalFiles     int                       `json:"total_files"`
	IndexedCount   int                       `json:"indexed_count"`
	SkippedCount   int                       `json:"skipped_count"`
	FailedCount    int                       `json:"failed_count"`
}

// FileCheckpoint contains information about a processed file
type FileCheckpoint struct {
	Path        string    `json:"path"`
	Hash        string    `json:"hash"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Status      string    `json:"status"` // "indexed", "skipped", "failed"
	ProcessedAt time.Time `json:"processed_at"`
}

// CheckpointManager manages indexing checkpoints
type CheckpointManager struct {
	checkpointDir string
	storeName     string
	checkpoint    *IndexCheckpoint
	enabled       bool
	saveInterval  time.Duration
	lastSaveTime  time.Time
	mu            sync.RWMutex // Protects checkpoint data
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(storeName, sourcePath string, enabled bool) *CheckpointManager {
	checkpointDir := filepath.Join(os.TempDir(), "hector-checkpoints")
	_ = os.MkdirAll(checkpointDir, 0755) // Ignore error - will fail on first save if needed

	return &CheckpointManager{
		checkpointDir: checkpointDir,
		storeName:     storeName,
		enabled:       enabled,
		saveInterval:  10 * time.Second, // Save every 10 seconds
		checkpoint: &IndexCheckpoint{
			Version:        "1.0",
			StoreName:      storeName,
			SourcePath:     sourcePath,
			StartTime:      time.Now(),
			LastUpdate:     time.Now(),
			ProcessedFiles: make(map[string]FileCheckpoint),
		},
	}
}

// LoadCheckpoint attempts to load an existing checkpoint
func (cm *CheckpointManager) LoadCheckpoint() (*IndexCheckpoint, error) {
	if !cm.enabled {
		return nil, nil
	}

	checkpointPath := cm.getCheckpointPath()
	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists
		}
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var checkpoint IndexCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	cm.checkpoint = &checkpoint
	return &checkpoint, nil
}

// SaveCheckpoint saves the current checkpoint
func (cm *CheckpointManager) SaveCheckpoint() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.RLock()
	// Throttle saves to avoid excessive I/O
	if time.Since(cm.lastSaveTime) < cm.saveInterval {
		cm.mu.RUnlock()
		return nil
	}

	cm.checkpoint.LastUpdate = time.Now()

	data, err := json.MarshalIndent(cm.checkpoint, "", "  ")
	cm.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	checkpointPath := cm.getCheckpointPath()
	if err := os.WriteFile(checkpointPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	cm.mu.Lock()
	cm.lastSaveTime = time.Now()
	cm.mu.Unlock()
	return nil
}

// RecordFile records a processed file in the checkpoint
func (cm *CheckpointManager) RecordFile(path string, size int64, modTime time.Time, status string) {
	if !cm.enabled {
		return
	}

	hash := cm.computeFileHash(path, size, modTime)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.checkpoint.ProcessedFiles[path] = FileCheckpoint{
		Path:        path,
		Hash:        hash,
		Size:        size,
		ModTime:     modTime,
		Status:      status,
		ProcessedAt: time.Now(),
	}

	// Update counters
	switch status {
	case "indexed":
		cm.checkpoint.IndexedCount++
	case "skipped":
		cm.checkpoint.SkippedCount++
	case "failed":
		cm.checkpoint.FailedCount++
	}
}

// ShouldProcessFile checks if a file should be processed (not in checkpoint or changed)
func (cm *CheckpointManager) ShouldProcessFile(path string, size int64, modTime time.Time) bool {
	if !cm.enabled {
		return true
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	fileCheckpoint, exists := cm.checkpoint.ProcessedFiles[path]
	if !exists {
		return true // File not in checkpoint
	}

	// Check if file has changed
	currentHash := cm.computeFileHash(path, size, modTime)
	if currentHash != fileCheckpoint.Hash {
		return true // File has changed
	}

	// Skip if previously indexed or skipped successfully
	return fileCheckpoint.Status == "failed"
}

// SetTotalFiles sets the total file count
func (cm *CheckpointManager) SetTotalFiles(total int) {
	if cm.enabled {
		cm.mu.Lock()
		cm.checkpoint.TotalFiles = total
		cm.mu.Unlock()
	}
}

// GetProcessedCount returns the number of processed files
func (cm *CheckpointManager) GetProcessedCount() int {
	if !cm.enabled {
		return 0
	}
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.checkpoint.ProcessedFiles)
}

// ClearCheckpoint removes the checkpoint file
func (cm *CheckpointManager) ClearCheckpoint() error {
	if !cm.enabled {
		return nil
	}

	checkpointPath := cm.getCheckpointPath()
	if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove checkpoint: %w", err)
	}

	return nil
}

// getCheckpointPath returns the path to the checkpoint file
func (cm *CheckpointManager) getCheckpointPath() string {
	// Create a unique filename based on store name and source path
	hash := md5.Sum([]byte(cm.storeName + ":" + cm.checkpoint.SourcePath))
	filename := fmt.Sprintf("checkpoint_%x.json", hash)
	return filepath.Join(cm.checkpointDir, filename)
}

// computeFileHash computes a hash for file identification
func (cm *CheckpointManager) computeFileHash(path string, size int64, modTime time.Time) string {
	data := fmt.Sprintf("%s:%d:%d", path, size, modTime.Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// FormatCheckpointInfo returns a human-readable checkpoint summary
func (cm *CheckpointManager) FormatCheckpointInfo(checkpoint *IndexCheckpoint) string {
	if checkpoint == nil {
		return ""
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	processed := len(checkpoint.ProcessedFiles)
	elapsed := time.Since(checkpoint.StartTime)

	return fmt.Sprintf("Found checkpoint: %d/%d files processed (%d indexed, %d skipped, %d failed) - %s elapsed",
		processed, checkpoint.TotalFiles,
		checkpoint.IndexedCount, checkpoint.SkippedCount, checkpoint.FailedCount,
		formatDuration(elapsed))
}
