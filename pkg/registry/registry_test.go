package registry

import (
	"fmt"
	"testing"
)

// TestItem is a simple struct for testing
type TestItem struct {
	ID   string
	Name string
}

func TestBaseRegistry_Register(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	tests := []struct {
		name    string
		item    TestItem
		wantErr bool
	}{
		{
			name: "register valid item",
			item: TestItem{
				ID:   "test-1",
				Name: "Test Item 1",
			},
			wantErr: false,
		},
		{
			name: "register item with empty name",
			item: TestItem{
				ID:   "",
				Name: "Test Item",
			},
			wantErr: true,
		},
		{
			name: "register duplicate item",
			item: TestItem{
				ID:   "test-1", // Same ID as first test
				Name: "Test Item 2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.item.ID, tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseRegistry.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseRegistry_Get(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Register a test item
	testItem := TestItem{
		ID:   "test-1",
		Name: "Test Item 1",
	}
	err := registry.Register("test-1", testItem)
	if err != nil {
		t.Fatalf("Failed to register test item: %v", err)
	}

	tests := []struct {
		name     string
		itemID   string
		wantItem TestItem
		wantOk   bool
	}{
		{
			name:     "get existing item",
			itemID:   "test-1",
			wantItem: testItem,
			wantOk:   true,
		},
		{
			name:     "get non-existing item",
			itemID:   "non-existing",
			wantItem: TestItem{},
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, ok := registry.Get(tt.itemID)
			if ok != tt.wantOk {
				t.Errorf("BaseRegistry.Get() ok = %v, want %v", ok, tt.wantOk)
			}
			if item.ID != tt.wantItem.ID {
				t.Errorf("BaseRegistry.Get() item.ID = %v, want %v", item.ID, tt.wantItem.ID)
			}
			if item.Name != tt.wantItem.Name {
				t.Errorf("BaseRegistry.Get() item.Name = %v, want %v", item.Name, tt.wantItem.Name)
			}
		})
	}
}

func TestBaseRegistry_List(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Initially empty
	items := registry.List()
	if len(items) != 0 {
		t.Errorf("BaseRegistry.List() length = %v, want %v", len(items), 0)
	}

	// Register multiple items
	testItems := []TestItem{
		{ID: "test-1", Name: "Test Item 1"},
		{ID: "test-2", Name: "Test Item 2"},
		{ID: "test-3", Name: "Test Item 3"},
	}

	for _, item := range testItems {
		err := registry.Register(item.ID, item)
		if err != nil {
			t.Fatalf("Failed to register item %s: %v", item.ID, err)
		}
	}

	// Check list
	items = registry.List()
	if len(items) != len(testItems) {
		t.Errorf("BaseRegistry.List() length = %v, want %v", len(items), len(testItems))
	}

	// Verify all items are present
	itemMap := make(map[string]TestItem)
	for _, item := range items {
		itemMap[item.ID] = item
	}

	for _, expectedItem := range testItems {
		if actualItem, exists := itemMap[expectedItem.ID]; !exists {
			t.Errorf("BaseRegistry.List() missing item %s", expectedItem.ID)
		} else if actualItem.Name != expectedItem.Name {
			t.Errorf("BaseRegistry.List() item %s name = %v, want %v", expectedItem.ID, actualItem.Name, expectedItem.Name)
		}
	}
}

func TestBaseRegistry_Remove(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Register a test item
	testItem := TestItem{
		ID:   "test-1",
		Name: "Test Item 1",
	}
	err := registry.Register("test-1", testItem)
	if err != nil {
		t.Fatalf("Failed to register test item: %v", err)
	}

	tests := []struct {
		name    string
		itemID  string
		wantErr bool
	}{
		{
			name:    "remove existing item",
			itemID:  "test-1",
			wantErr: false,
		},
		{
			name:    "remove non-existing item",
			itemID:  "non-existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Remove(tt.itemID)
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseRegistry.Remove() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify item is actually removed
			if !tt.wantErr {
				_, exists := registry.Get(tt.itemID)
				if exists {
					t.Errorf("BaseRegistry.Remove() item %s still exists after removal", tt.itemID)
				}
			}
		})
	}
}

func TestBaseRegistry_Count(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Initially empty
	if count := registry.Count(); count != 0 {
		t.Errorf("BaseRegistry.Count() = %v, want %v", count, 0)
	}

	// Register items
	testItems := []TestItem{
		{ID: "test-1", Name: "Test Item 1"},
		{ID: "test-2", Name: "Test Item 2"},
	}

	for i, item := range testItems {
		err := registry.Register(item.ID, item)
		if err != nil {
			t.Fatalf("Failed to register item %s: %v", item.ID, err)
		}

		expectedCount := i + 1
		if count := registry.Count(); count != expectedCount {
			t.Errorf("BaseRegistry.Count() = %v, want %v", count, expectedCount)
		}
	}
}

func TestBaseRegistry_Clear(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Register some items
	testItems := []TestItem{
		{ID: "test-1", Name: "Test Item 1"},
		{ID: "test-2", Name: "Test Item 2"},
	}

	for _, item := range testItems {
		err := registry.Register(item.ID, item)
		if err != nil {
			t.Fatalf("Failed to register item %s: %v", item.ID, err)
		}
	}

	// Verify items exist
	if count := registry.Count(); count != len(testItems) {
		t.Errorf("BaseRegistry.Count() before clear = %v, want %v", count, len(testItems))
	}

	// Clear registry
	registry.Clear()

	// Verify registry is empty
	if count := registry.Count(); count != 0 {
		t.Errorf("BaseRegistry.Count() after clear = %v, want %v", count, 0)
	}

	items := registry.List()
	if len(items) != 0 {
		t.Errorf("BaseRegistry.List() after clear length = %v, want %v", len(items), 0)
	}

	// Verify individual items are gone
	for _, item := range testItems {
		_, exists := registry.Get(item.ID)
		if exists {
			t.Errorf("BaseRegistry.Get() item %s still exists after clear", item.ID)
		}
	}
}

func TestBaseRegistry_Concurrency(t *testing.T) {
	registry := NewBaseRegistry[TestItem]()

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Register items
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			item := TestItem{
				ID:   fmt.Sprintf("concurrent-%d", i),
				Name: fmt.Sprintf("Concurrent Item %d", i),
			}
			_ = registry.Register(item.ID, item)
		}
	}()

	// Goroutine 2: Read items
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			registry.Get(fmt.Sprintf("concurrent-%d", i))
			registry.Count()
			registry.List()
		}
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	if count := registry.Count(); count != 100 {
		t.Errorf("BaseRegistry.Count() after concurrent access = %v, want %v", count, 100)
	}
}
