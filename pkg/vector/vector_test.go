package vector_test

import (
	"context"
	"testing"

	"github.com/verikod/hector/pkg/vector"
)

// =============================================================================
// NilProvider Tests
// =============================================================================

func TestNilProvider_ImplementsProvider(t *testing.T) {
	var _ vector.Provider = vector.NilProvider{}
}

func TestNilProvider_Name(t *testing.T) {
	p := vector.NilProvider{}
	if p.Name() != "nil" {
		t.Errorf("Name() = %q, want nil", p.Name())
	}
}

func TestNilProvider_Upsert(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	err := p.Upsert(ctx, "collection", "id", []float32{0.1, 0.2}, nil)
	if err != nil {
		t.Errorf("Upsert() error = %v", err)
	}
}

func TestNilProvider_Search(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	results, err := p.Search(ctx, "collection", []float32{0.1, 0.2}, 10)
	if err != nil {
		t.Errorf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search() returned %d results, want 0", len(results))
	}
}

func TestNilProvider_SearchWithFilter(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	results, err := p.SearchWithFilter(ctx, "collection", []float32{0.1}, 10, nil)
	if err != nil {
		t.Errorf("SearchWithFilter() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchWithFilter() returned %d results, want 0", len(results))
	}
}

func TestNilProvider_Delete(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	err := p.Delete(ctx, "collection", "id")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestNilProvider_DeleteByFilter(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	err := p.DeleteByFilter(ctx, "collection", map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("DeleteByFilter() error = %v", err)
	}
}

func TestNilProvider_CreateCollection(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	err := p.CreateCollection(ctx, "collection", 768)
	if err != nil {
		t.Errorf("CreateCollection() error = %v", err)
	}
}

func TestNilProvider_DeleteCollection(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	err := p.DeleteCollection(ctx, "collection")
	if err != nil {
		t.Errorf("DeleteCollection() error = %v", err)
	}
}

func TestNilProvider_Count(t *testing.T) {
	p := vector.NilProvider{}
	ctx := context.Background()

	count, err := p.Count(ctx, "collection")
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

func TestNilProvider_Close(t *testing.T) {
	p := vector.NilProvider{}

	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// =============================================================================
// Registry Tests
// =============================================================================

func TestNewRegistry(t *testing.T) {
	r := vector.NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if len(r.List()) != 0 {
		t.Errorf("New registry should have 0 providers")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := vector.NewRegistry()
	p := vector.NilProvider{}

	err := r.Register("test", p)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify it's registered
	names := r.List()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("List() = %v, want [test]", names)
	}
}

func TestRegistry_Register_Errors(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		r := vector.NewRegistry()
		err := r.Register("", vector.NilProvider{})
		if err == nil {
			t.Error("Register() should error on empty name")
		}
	})

	t.Run("nil provider", func(t *testing.T) {
		r := vector.NewRegistry()
		err := r.Register("test", nil)
		if err == nil {
			t.Error("Register() should error on nil provider")
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		r := vector.NewRegistry()
		_ = r.Register("test", vector.NilProvider{})

		err := r.Register("test", vector.NilProvider{})
		if err == nil {
			t.Error("Register() should error on duplicate name")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	r := vector.NewRegistry()
	_ = r.Register("test", vector.NilProvider{})

	t.Run("existing provider", func(t *testing.T) {
		p, ok := r.Get("test")
		if !ok {
			t.Error("Get() should find registered provider")
		}
		if p.Name() != "nil" {
			t.Errorf("Got wrong provider type")
		}
	})

	t.Run("nonexistent provider", func(t *testing.T) {
		_, ok := r.Get("nonexistent")
		if ok {
			t.Error("Get() should return false for nonexistent provider")
		}
	})
}

func TestRegistry_MustGet(t *testing.T) {
	r := vector.NewRegistry()
	_ = r.Register("test", vector.NilProvider{})

	t.Run("existing provider", func(t *testing.T) {
		p := r.MustGet("test")
		if p == nil {
			t.Error("MustGet() should return provider")
		}
	})

	t.Run("panics on nonexistent", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet() should panic for nonexistent provider")
			}
		}()
		r.MustGet("nonexistent")
	})
}

func TestRegistry_List(t *testing.T) {
	r := vector.NewRegistry()
	_ = r.Register("a", vector.NilProvider{})
	_ = r.Register("b", vector.NilProvider{})
	_ = r.Register("c", vector.NilProvider{})

	names := r.List()
	if len(names) != 3 {
		t.Errorf("List() returned %d names, want 3", len(names))
	}
}

func TestRegistry_Close(t *testing.T) {
	r := vector.NewRegistry()
	_ = r.Register("test", vector.NilProvider{})

	err := r.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// After close, list should be empty
	if len(r.List()) != 0 {
		t.Error("Registry should be empty after Close()")
	}
}

// =============================================================================
// ProviderConfig Tests
// =============================================================================

func TestProviderConfig_SetDefaults(t *testing.T) {
	t.Run("empty config gets chromem default", func(t *testing.T) {
		cfg := &vector.ProviderConfig{}
		cfg.SetDefaults()

		if cfg.Type != vector.ProviderChromem {
			t.Errorf("Type = %q, want chromem", cfg.Type)
		}
		if cfg.Chromem == nil {
			t.Error("Chromem config should be initialized")
		}
	})

	t.Run("preserves existing type", func(t *testing.T) {
		cfg := &vector.ProviderConfig{Type: vector.ProviderQdrant}
		cfg.SetDefaults()

		if cfg.Type != vector.ProviderQdrant {
			t.Error("Type should be preserved")
		}
	})
}

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *vector.ProviderConfig
		wantErr bool
	}{
		{
			name:    "chromem is valid without config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderChromem},
			wantErr: false,
		},
		{
			name:    "empty type is invalid",
			cfg:     &vector.ProviderConfig{},
			wantErr: true,
		},
		{
			name:    "unknown type is invalid",
			cfg:     &vector.ProviderConfig{Type: "unknown"},
			wantErr: true,
		},
		{
			name:    "qdrant requires config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderQdrant},
			wantErr: true,
		},
		{
			name: "qdrant requires host",
			cfg: &vector.ProviderConfig{
				Type:   vector.ProviderQdrant,
				Qdrant: &vector.QdrantConfig{},
			},
			wantErr: true,
		},
		{
			name:    "pinecone requires config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderPinecone},
			wantErr: true,
		},
		{
			name:    "weaviate requires config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderWeaviate},
			wantErr: true,
		},
		{
			name:    "milvus requires config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderMilvus},
			wantErr: true,
		},
		{
			name:    "chroma requires config",
			cfg:     &vector.ProviderConfig{Type: vector.ProviderChroma},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// NewProvider Tests
// =============================================================================

func TestNewProvider_NilConfig(t *testing.T) {
	p, err := vector.NewProvider(nil)
	if err != nil {
		t.Fatalf("NewProvider(nil) error = %v", err)
	}
	if p.Name() != "nil" {
		t.Errorf("NewProvider(nil) should return NilProvider")
	}
}

func TestNewProvider_UnknownType(t *testing.T) {
	cfg := &vector.ProviderConfig{Type: "unknown-type"}
	_, err := vector.NewProvider(cfg)
	if err == nil {
		t.Error("NewProvider() should error on unknown type")
	}
}

// =============================================================================
// ProviderType Constants Tests
// =============================================================================

func TestProviderTypeConstants(t *testing.T) {
	tests := []struct {
		providerType vector.ProviderType
		expected     string
	}{
		{vector.ProviderChromem, "chromem"},
		{vector.ProviderQdrant, "qdrant"},
		{vector.ProviderChroma, "chroma"},
		{vector.ProviderPinecone, "pinecone"},
		{vector.ProviderMilvus, "milvus"},
		{vector.ProviderWeaviate, "weaviate"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.providerType) != tt.expected {
				t.Errorf("ProviderType = %q, want %q", tt.providerType, tt.expected)
			}
		})
	}
}

// =============================================================================
// Result Tests
// =============================================================================

func TestResult_Fields(t *testing.T) {
	result := vector.Result{
		ID:       "doc-1",
		Score:    0.95,
		Content:  "Test content",
		Vector:   []float32{0.1, 0.2, 0.3},
		Metadata: map[string]any{"key": "value"},
	}

	if result.ID != "doc-1" {
		t.Errorf("ID = %q", result.ID)
	}
	if result.Score != 0.95 {
		t.Errorf("Score = %f", result.Score)
	}
	if result.Content != "Test content" {
		t.Errorf("Content = %q", result.Content)
	}
	if len(result.Vector) != 3 {
		t.Errorf("Vector length = %d", len(result.Vector))
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestConfig_Fields(t *testing.T) {
	cfg := vector.Config{
		Type:       "chromem",
		Collection: "test-collection",
	}

	if cfg.Type != "chromem" {
		t.Errorf("Type = %q", cfg.Type)
	}
	if cfg.Collection != "test-collection" {
		t.Errorf("Collection = %q", cfg.Collection)
	}
}
