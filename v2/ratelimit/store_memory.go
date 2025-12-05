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

package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// usageKey uniquely identifies a usage record.
type usageKey struct {
	Scope      Scope
	Identifier string
	LimitType  LimitType
	Window     TimeWindow
}

// usageRecord stores usage data.
type usageRecord struct {
	Amount    int64
	WindowEnd time.Time
}

// MemoryStore is an in-memory implementation of Store.
// It is thread-safe and suitable for development, testing, and single-instance deployments.
type MemoryStore struct {
	data map[usageKey]*usageRecord
	mu   sync.RWMutex
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[usageKey]*usageRecord),
	}
}

// GetUsage gets current usage for a specific limit.
func (s *MemoryStore) GetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow) (int64, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := usageKey{
		Scope:      scope,
		Identifier: identifier,
		LimitType:  limitType,
		Window:     window,
	}

	record, exists := s.data[key]
	if !exists {
		// No usage yet, return 0 with future window
		return 0, time.Now().Add(window.Duration()), nil
	}

	// Check if window has expired
	now := time.Now()
	if record.WindowEnd.Before(now) {
		// Window expired, return 0 with new window
		return 0, now.Add(window.Duration()), nil
	}

	return record.Amount, record.WindowEnd, nil
}

// IncrementUsage increments usage for a specific limit.
func (s *MemoryStore) IncrementUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64) (int64, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := usageKey{
		Scope:      scope,
		Identifier: identifier,
		LimitType:  limitType,
		Window:     window,
	}

	now := time.Now()
	record, exists := s.data[key]

	if !exists {
		// Create new record
		record = &usageRecord{
			Amount:    amount,
			WindowEnd: now.Add(window.Duration()),
		}
		s.data[key] = record
		return record.Amount, record.WindowEnd, nil
	}

	// Check if window has expired
	if record.WindowEnd.Before(now) {
		// Reset window
		record.Amount = amount
		record.WindowEnd = now.Add(window.Duration())
	} else {
		// Increment existing
		record.Amount += amount
	}

	return record.Amount, record.WindowEnd, nil
}

// SetUsage sets usage for a specific limit.
func (s *MemoryStore) SetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64, windowEnd time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := usageKey{
		Scope:      scope,
		Identifier: identifier,
		LimitType:  limitType,
		Window:     window,
	}

	s.data[key] = &usageRecord{
		Amount:    amount,
		WindowEnd: windowEnd,
	}

	return nil
}

// DeleteUsage deletes usage records for an identifier.
func (s *MemoryStore) DeleteUsage(ctx context.Context, scope Scope, identifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and delete all keys matching the identifier
	for key := range s.data {
		if key.Scope == scope && key.Identifier == identifier {
			delete(s.data, key)
		}
	}

	return nil
}

// DeleteExpired deletes expired usage records.
func (s *MemoryStore) DeleteExpired(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and delete all expired records
	for key, record := range s.data {
		if record.WindowEnd.Before(before) {
			delete(s.data, key)
		}
	}

	return nil
}

// Close closes the store.
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear all data
	s.data = make(map[usageKey]*usageRecord)
	return nil
}

// Size returns the number of records in the store (for testing).
func (s *MemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Dump returns all records as a map (for debugging).
func (s *MemoryStore) Dump() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for key, record := range s.data {
		keyStr := fmt.Sprintf("%s:%s:%s:%s", key.Scope, key.Identifier, key.LimitType, key.Window)
		result[keyStr] = map[string]interface{}{
			"amount":     record.Amount,
			"window_end": record.WindowEnd,
		}
	}
	return result
}
