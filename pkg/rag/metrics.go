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
	"sync"
	"sync/atomic"
	"time"
)

// IndexMetrics tracks document store indexing metrics.
//
// Thread-safe for concurrent access during indexing.
type IndexMetrics struct {
	storeName string

	// Counters
	totalDocs   int64
	indexedDocs int64
	skippedDocs int64
	errorDocs   int64

	// Timing
	startTime time.Time
	endTime   time.Time

	// Search metrics
	searchCount       int64
	searchLatencySum  int64 // nanoseconds
	searchLatencyMax  int64
	lastSearchLatency int64

	mu sync.RWMutex
}

// NewIndexMetrics creates a new metrics tracker.
func NewIndexMetrics(storeName string) *IndexMetrics {
	return &IndexMetrics{
		storeName: storeName,
	}
}

// Reset clears all metrics.
func (m *IndexMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalDocs = 0
	m.indexedDocs = 0
	m.skippedDocs = 0
	m.errorDocs = 0
	m.startTime = time.Time{}
	m.endTime = time.Time{}
	m.searchCount = 0
	m.searchLatencySum = 0
	m.searchLatencyMax = 0
	m.lastSearchLatency = 0
}

// SetStartTime sets the indexing start time.
func (m *IndexMetrics) SetStartTime(t time.Time) {
	m.mu.Lock()
	m.startTime = t
	m.mu.Unlock()
}

// SetEndTime sets the indexing end time.
func (m *IndexMetrics) SetEndTime(t time.Time) {
	m.mu.Lock()
	m.endTime = t
	m.mu.Unlock()
}

// IncrementTotal increments total document count.
func (m *IndexMetrics) IncrementTotal() {
	atomic.AddInt64(&m.totalDocs, 1)
}

// IncrementIndexed increments indexed document count.
func (m *IndexMetrics) IncrementIndexed() {
	atomic.AddInt64(&m.indexedDocs, 1)
}

// IncrementSkipped increments skipped document count.
func (m *IndexMetrics) IncrementSkipped() {
	atomic.AddInt64(&m.skippedDocs, 1)
}

// IncrementErrors increments error count.
func (m *IndexMetrics) IncrementErrors() {
	atomic.AddInt64(&m.errorDocs, 1)
}

// RecordSearch records a search operation with latency.
func (m *IndexMetrics) RecordSearch(latency time.Duration) {
	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.searchCount, 1)
	atomic.AddInt64(&m.searchLatencySum, latencyNs)
	atomic.StoreInt64(&m.lastSearchLatency, latencyNs)

	// Update max latency (CAS loop for atomic max)
	for {
		current := atomic.LoadInt64(&m.searchLatencyMax)
		if latencyNs <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.searchLatencyMax, current, latencyNs) {
			break
		}
	}
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *IndexMetrics) Snapshot() IndexMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := atomic.LoadInt64(&m.totalDocs)
	indexed := atomic.LoadInt64(&m.indexedDocs)
	searchCount := atomic.LoadInt64(&m.searchCount)
	searchLatencySum := atomic.LoadInt64(&m.searchLatencySum)

	var docsPerSec float64
	var avgSearchLatency time.Duration

	if !m.startTime.IsZero() {
		endTime := m.endTime
		if endTime.IsZero() {
			endTime = time.Now()
		}
		elapsed := endTime.Sub(m.startTime).Seconds()
		if elapsed > 0 {
			docsPerSec = float64(indexed) / elapsed
		}
	}

	if searchCount > 0 {
		avgSearchLatency = time.Duration(searchLatencySum / searchCount)
	}

	return IndexMetricsSnapshot{
		StoreName:         m.storeName,
		TotalDocs:         total,
		IndexedDocs:       indexed,
		SkippedDocs:       atomic.LoadInt64(&m.skippedDocs),
		ErrorDocs:         atomic.LoadInt64(&m.errorDocs),
		DocsPerSecond:     docsPerSec,
		StartTime:         m.startTime,
		EndTime:           m.endTime,
		SearchCount:       searchCount,
		AvgSearchLatency:  avgSearchLatency,
		MaxSearchLatency:  time.Duration(atomic.LoadInt64(&m.searchLatencyMax)),
		LastSearchLatency: time.Duration(atomic.LoadInt64(&m.lastSearchLatency)),
	}
}

// IndexMetricsSnapshot is a point-in-time copy of metrics.
type IndexMetricsSnapshot struct {
	StoreName         string        `json:"store_name"`
	TotalDocs         int64         `json:"total_docs"`
	IndexedDocs       int64         `json:"indexed_docs"`
	SkippedDocs       int64         `json:"skipped_docs"`
	ErrorDocs         int64         `json:"error_docs"`
	DocsPerSecond     float64       `json:"docs_per_second"`
	StartTime         time.Time     `json:"start_time,omitempty"`
	EndTime           time.Time     `json:"end_time,omitempty"`
	SearchCount       int64         `json:"search_count"`
	AvgSearchLatency  time.Duration `json:"avg_search_latency_ns"`
	MaxSearchLatency  time.Duration `json:"max_search_latency_ns"`
	LastSearchLatency time.Duration `json:"last_search_latency_ns"`
}

// SearchMetrics tracks search engine metrics.
type SearchMetrics struct {
	engineName string

	// Search counters
	totalSearches  int64
	successfulHits int64
	emptyResults   int64

	// Latency tracking
	latencySum int64 // nanoseconds
	latencyMax int64
	latencyMin int64

	// Feature usage
	hydeEnabled       int64
	rerankEnabled     int64
	multiQueryEnabled int64

	//nolint:unused // Reserved for future use
	mu sync.RWMutex
}

// NewSearchMetrics creates a new search metrics tracker.
func NewSearchMetrics(engineName string) *SearchMetrics {
	return &SearchMetrics{
		engineName: engineName,
		latencyMin: int64(^uint64(0) >> 1), // Max int64
	}
}

// RecordSearch records a search operation.
func (m *SearchMetrics) RecordSearch(latency time.Duration, resultCount int, opts *SearchOptions) {
	latencyNs := latency.Nanoseconds()

	atomic.AddInt64(&m.totalSearches, 1)
	atomic.AddInt64(&m.latencySum, latencyNs)

	if resultCount > 0 {
		atomic.AddInt64(&m.successfulHits, 1)
	} else {
		atomic.AddInt64(&m.emptyResults, 1)
	}

	// Update max latency
	for {
		current := atomic.LoadInt64(&m.latencyMax)
		if latencyNs <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.latencyMax, current, latencyNs) {
			break
		}
	}

	// Update min latency
	for {
		current := atomic.LoadInt64(&m.latencyMin)
		if latencyNs >= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.latencyMin, current, latencyNs) {
			break
		}
	}

	// Track feature usage
	if opts != nil {
		if opts.EnableHyDE {
			atomic.AddInt64(&m.hydeEnabled, 1)
		}
		if opts.EnableRerank {
			atomic.AddInt64(&m.rerankEnabled, 1)
		}
		if opts.EnableMultiQuery {
			atomic.AddInt64(&m.multiQueryEnabled, 1)
		}
	}
}

// Snapshot returns a point-in-time copy of search metrics.
func (m *SearchMetrics) Snapshot() SearchMetricsSnapshot {
	total := atomic.LoadInt64(&m.totalSearches)
	latencySum := atomic.LoadInt64(&m.latencySum)
	latencyMin := atomic.LoadInt64(&m.latencyMin)

	var avgLatency time.Duration
	if total > 0 {
		avgLatency = time.Duration(latencySum / total)
	}

	// Handle case where no searches have occurred
	if latencyMin == int64(^uint64(0)>>1) {
		latencyMin = 0
	}

	return SearchMetricsSnapshot{
		EngineName:      m.engineName,
		TotalSearches:   total,
		SuccessfulHits:  atomic.LoadInt64(&m.successfulHits),
		EmptyResults:    atomic.LoadInt64(&m.emptyResults),
		AvgLatency:      avgLatency,
		MaxLatency:      time.Duration(atomic.LoadInt64(&m.latencyMax)),
		MinLatency:      time.Duration(latencyMin),
		HyDEUsage:       atomic.LoadInt64(&m.hydeEnabled),
		RerankUsage:     atomic.LoadInt64(&m.rerankEnabled),
		MultiQueryUsage: atomic.LoadInt64(&m.multiQueryEnabled),
	}
}

// SearchMetricsSnapshot is a point-in-time copy of search metrics.
type SearchMetricsSnapshot struct {
	EngineName      string        `json:"engine_name"`
	TotalSearches   int64         `json:"total_searches"`
	SuccessfulHits  int64         `json:"successful_hits"`
	EmptyResults    int64         `json:"empty_results"`
	AvgLatency      time.Duration `json:"avg_latency_ns"`
	MaxLatency      time.Duration `json:"max_latency_ns"`
	MinLatency      time.Duration `json:"min_latency_ns"`
	HyDEUsage       int64         `json:"hyde_usage"`
	RerankUsage     int64         `json:"rerank_usage"`
	MultiQueryUsage int64         `json:"multi_query_usage"`
}
