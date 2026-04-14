package native

import (
	"context"
	"sync"

	"github.com/a2aproject/a2a-go/a2a"
)

// CollectingEventQueue collects A2A events for non-streaming execution.
// It implements a2asrv/eventqueue.Queue interface.
type CollectingEventQueue struct {
	events []a2a.Event
	mu     sync.Mutex
	closed bool
}

// NewCollectingEventQueue creates a new collector.
func NewCollectingEventQueue() *CollectingEventQueue {
	return &CollectingEventQueue{
		events: make([]a2a.Event, 0),
	}
}

// Write adds an event to the collection.
func (q *CollectingEventQueue) Write(ctx context.Context, event a2a.Event) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.events = append(q.events, event)
	return nil
}

// Read returns the next event (for interface compatibility).
// This is typically not used - use Events() instead.
func (q *CollectingEventQueue) Read(ctx context.Context) (a2a.Event, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.events) == 0 {
		return nil, context.Canceled
	}

	event := q.events[0]
	q.events = q.events[1:]
	return event, nil
}

// Close marks the queue as closed.
func (q *CollectingEventQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}

// Events returns all collected events.
func (q *CollectingEventQueue) Events() []a2a.Event {
	q.mu.Lock()
	defer q.mu.Unlock()
	result := make([]a2a.Event, len(q.events))
	copy(result, q.events)
	return result
}
