package registry

import (
	"fmt"
	"sync"
)

type Registry[T any] interface {
	Register(name string, item T) error
	Get(name string) (T, bool)
	List() []T
	Remove(name string) error
	Count() int
	Clear()
}

type BaseRegistry[T any] struct {
	mu    sync.RWMutex
	items map[string]T
}

func NewBaseRegistry[T any]() *BaseRegistry[T] {
	return &BaseRegistry[T]{
		items: make(map[string]T),
	}
}

func (r *BaseRegistry[T]) Register(name string, item T) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[name]; exists {
		return fmt.Errorf("item with name '%s' already registered", name)
	}

	r.items[name] = item
	return nil
}

func (r *BaseRegistry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.items[name]
	return item, exists
}

func (r *BaseRegistry[T]) List() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]T, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

func (r *BaseRegistry[T]) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[name]; !exists {
		return fmt.Errorf("item '%s' not found", name)
	}

	delete(r.items, name)
	return nil
}

func (r *BaseRegistry[T]) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.items)
}

func (r *BaseRegistry[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[string]T)
}
