package context

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressTracker tracks indexing progress with real-time statistics
type ProgressTracker struct {
	// Counters (atomic for thread-safe increments)
	totalFiles     int64
	processedFiles int64
	indexedFiles   int64
	skippedFiles   int64
	failedFiles    int64
	deletedFiles   int64

	// Current state
	currentFile   string
	currentFileMu sync.RWMutex

	// Timing
	startTime      time.Time
	lastUpdateTime time.Time

	// Display settings
	enabled         bool
	displayInterval time.Duration
	verbose         bool

	// Stop channel and state
	stopChan chan struct{}
	doneChan chan struct{}
	running  int32 // Atomic flag to track if running
	mu       sync.Mutex
	printMu  sync.Mutex // Mutex for print operations to prevent interleaving
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(enabled bool, verbose bool) *ProgressTracker {
	return &ProgressTracker{
		enabled:         enabled,
		verbose:         verbose,
		displayInterval: 1 * time.Second, // Update every second
		startTime:       time.Now(),
		lastUpdateTime:  time.Now(),
		stopChan:        make(chan struct{}),
		doneChan:        make(chan struct{}),
		running:         0,
	}
}

// Start begins the progress display loop
func (pt *ProgressTracker) Start() {
	if !pt.enabled {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Check if already running
	if atomic.LoadInt32(&pt.running) == 1 {
		return // Already running
	}

	// Reset channels
	pt.stopChan = make(chan struct{})
	pt.doneChan = make(chan struct{})

	atomic.StoreInt32(&pt.running, 1)
	go pt.displayLoop()
}

// Stop stops the progress display
func (pt *ProgressTracker) Stop() {
	if !pt.enabled {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Check if already stopped
	if atomic.LoadInt32(&pt.running) == 0 {
		return // Already stopped
	}

	atomic.StoreInt32(&pt.running, 0)
	close(pt.stopChan)
	<-pt.doneChan // Wait for display loop to finish

	// Print one final progress update to show 100%
	pt.printProgress()

	// Print final summary
	pt.printFinalSummary()
}

// SetTotalFiles sets the total number of files to process
func (pt *ProgressTracker) SetTotalFiles(total int64) {
	atomic.StoreInt64(&pt.totalFiles, total)
}

// SetCurrentFile sets the currently processing file
func (pt *ProgressTracker) SetCurrentFile(filename string) {
	if !pt.enabled {
		return
	}

	pt.currentFileMu.Lock()
	pt.currentFile = filename
	pt.currentFileMu.Unlock()
}

// IncrementProcessed increments the processed files counter
func (pt *ProgressTracker) IncrementProcessed() {
	atomic.AddInt64(&pt.processedFiles, 1)
}

// IncrementIndexed increments the indexed files counter
func (pt *ProgressTracker) IncrementIndexed() {
	atomic.AddInt64(&pt.indexedFiles, 1)
}

// IncrementSkipped increments the skipped files counter
func (pt *ProgressTracker) IncrementSkipped() {
	atomic.AddInt64(&pt.skippedFiles, 1)
}

// IncrementFailed increments the failed files counter
func (pt *ProgressTracker) IncrementFailed() {
	atomic.AddInt64(&pt.failedFiles, 1)
}

// IncrementDeleted increments the deleted files counter
func (pt *ProgressTracker) IncrementDeleted() {
	atomic.AddInt64(&pt.deletedFiles, 1)
}

// GetStats returns current statistics
func (pt *ProgressTracker) GetStats() ProgressStats {
	pt.currentFileMu.RLock()
	currentFile := pt.currentFile
	pt.currentFileMu.RUnlock()

	return ProgressStats{
		TotalFiles:     atomic.LoadInt64(&pt.totalFiles),
		ProcessedFiles: atomic.LoadInt64(&pt.processedFiles),
		IndexedFiles:   atomic.LoadInt64(&pt.indexedFiles),
		SkippedFiles:   atomic.LoadInt64(&pt.skippedFiles),
		FailedFiles:    atomic.LoadInt64(&pt.failedFiles),
		DeletedFiles:   atomic.LoadInt64(&pt.deletedFiles),
		CurrentFile:    currentFile,
		ElapsedTime:    time.Since(pt.startTime),
	}
}

// displayLoop continuously displays progress updates
func (pt *ProgressTracker) displayLoop() {
	defer close(pt.doneChan)

	ticker := time.NewTicker(pt.displayInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pt.printProgress()
		case <-pt.stopChan:
			return
		}
	}
}

// printProgress prints the current progress
func (pt *ProgressTracker) printProgress() {
	// Use mutex to ensure only one progress update prints at a time
	// This prevents interleaved output that can cause newlines
	pt.printMu.Lock()
	defer pt.printMu.Unlock()
	
	stats := pt.GetStats()

	if stats.TotalFiles == 0 {
		// Don't display if we haven't discovered files yet
		return
	}

	percentage := float64(stats.ProcessedFiles) / float64(stats.TotalFiles) * 100
	elapsed := stats.ElapsedTime

	// Calculate ETA
	var eta time.Duration
	var filesPerSec float64
	if stats.ProcessedFiles > 0 && elapsed.Seconds() > 0 {
		filesPerSec = float64(stats.ProcessedFiles) / elapsed.Seconds()
		remaining := stats.TotalFiles - stats.ProcessedFiles
		if filesPerSec > 0 {
			eta = time.Duration(float64(remaining)/filesPerSec) * time.Second
		}
	}

	// Build progress bar
	barWidth := 30
	filled := int(percentage / 100 * float64(barWidth))
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	// Format output - print on new line so logs don't overwrite it
	output := fmt.Sprintf("📊 [%s] %.1f%% | %d/%d files",
		bar, percentage, stats.ProcessedFiles, stats.TotalFiles)

	if filesPerSec > 0 {
		output += fmt.Sprintf(" | %.1f files/s", filesPerSec)
	}

	if eta > 0 {
		output += fmt.Sprintf(" | ETA: %s", formatDuration(eta))
	}

	// Show errors and failures in progress line
	if stats.FailedFiles > 0 {
		output += fmt.Sprintf(" | ❌ %d failed", stats.FailedFiles)
	}

	if pt.verbose && stats.CurrentFile != "" {
		// Truncate filename if too long
		displayFile := stats.CurrentFile
		maxLen := 60
		if len(displayFile) > maxLen {
			displayFile = "..." + displayFile[len(displayFile)-maxLen+3:]
		}
		output += fmt.Sprintf(" | 📄 %s", displayFile)
	}

	// Print inline with carriage return to overwrite previous line
	// Use ANSI escape code to clear to end of line for better reliability
	// Use raw Write to avoid any buffering issues with fmt functions
	os.Stdout.WriteString("\r\033[K" + output)
	
	// Flush stdout to ensure the output is displayed immediately
	os.Stdout.Sync()
}

// printFinalSummary prints the final summary
func (pt *ProgressTracker) printFinalSummary() {
	stats := pt.GetStats()

	fmt.Print("\r\033[K") // Clear line
	fmt.Println("\n✅ Indexing Complete!")
	fmt.Printf("   Total:   %d files\n", stats.TotalFiles)
	fmt.Printf("   Indexed: %d files\n", stats.IndexedFiles)

	if stats.SkippedFiles > 0 {
		fmt.Printf("   Skipped: %d files\n", stats.SkippedFiles)
	}

	if stats.DeletedFiles > 0 {
		fmt.Printf("   🗑️  Deleted: %d files\n", stats.DeletedFiles)
	}

	if stats.FailedFiles > 0 {
		fmt.Printf("   ❌ Failed: %d files\n", stats.FailedFiles)
	}

	elapsed := stats.ElapsedTime
	fmt.Printf("   Time:    %s\n", formatDuration(elapsed))

	if stats.ProcessedFiles > 0 && elapsed.Seconds() > 0 {
		filesPerSec := float64(stats.ProcessedFiles) / elapsed.Seconds()
		fmt.Printf("   Speed:   %.1f files/s\n", filesPerSec)
	}
	fmt.Println()
}

// ProgressStats contains progress statistics
type ProgressStats struct {
	TotalFiles     int64
	ProcessedFiles int64
	IndexedFiles   int64
	SkippedFiles   int64
	FailedFiles    int64
	DeletedFiles   int64
	CurrentFile    string
	ElapsedTime    time.Duration
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}
