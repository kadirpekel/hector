package memory

import (
	"fmt"

	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/embedder"
	"github.com/verikod/hector/pkg/model"
	// "github.com/verikod/hector/pkg/rag" // Temporarily disabled during config refactoring
)

// NewIndexServiceFromConfig creates an IndexService based on configuration.
//
// Architecture (derived from legacy Hector):
//
//	┌─────────────────────────────────────────────────────────────┐
//	│   LAYER 3: IndexService (search index)                      │
//	│   - keyword: Simple word matching (default)                 │
//	│   - vector: Semantic search using embeddings                │
//	│   - CAN BE REBUILT from session.Service                     │
//	├─────────────────────────────────────────────────────────────┤
//	│   LAYER 2: session.Service (source of truth)                │
//	│   - SQL storage for all events                              │
//	│   - THIS IS THE SOURCE OF TRUTH                             │
//	├─────────────────────────────────────────────────────────────┤
//	│   LAYER 1: WorkingMemoryStrategy (context window)           │
//	│   - Ephemeral runtime cache                                 │
//	│   - Filters events for LLM context                          │
//	└─────────────────────────────────────────────────────────────┘
//
// Example config:
//
//	vector_stores:
//	  myvector:
//	    type: chromem
//	    persist_path: .hector/vectors
//
//	embedders:
//	  default:
//	    provider: openai
//	    model: text-embedding-3-small
//	    api_key: ${OPENAI_API_KEY}
//
//	storage:
//	  memory:
//	    backend: vector
//	    embedder: default
//	    vector_store: myvector
//
// NewIndexServiceFromConfig was the old interface that used config.Config.
// It has been removed during config refactoring.
func NewIndexServiceFromConfig(cfg interface{}, embedders map[string]embedder.Embedder) (IndexService, error) {
	// Temporarily disabled during config refactoring
	return nil, nil
	/*
		// Only create index service when memory is explicitly configured (opt-in).
		// Without explicit config, memory features are disabled.
		if cfg == nil || cfg.Memory == nil {
			return nil, nil
		}

		memCfg := cfg.Memory
		memCfg.SetDefaults()

		if err := memCfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid memory config: %w", err)
		}

		switch {
		case memCfg.IsKeyword():
			return NewKeywordIndexService(), nil

		case memCfg.IsVector():
			// Get embedder reference
			emb, ok := embedders[memCfg.Embedder]
			if !ok {
				return nil, fmt.Errorf("embedder %q not found (referenced by memory)", memCfg.Embedder)
			}

			// Get vector store config by reference
			vsCfg, ok := cfg.VectorStores[memCfg.VectorStore]
			if !ok {
				return nil, fmt.Errorf("vector_store %q not found (referenced by memory)", memCfg.VectorStore)
			}

			provider, err := rag.NewVectorProviderFromConfig(vsCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create vector provider: %w", err)
			}

			return NewVectorIndexService(VectorIndexConfig{
				Provider: provider,
				Embedder: emb,
			})

		default:
			return nil, fmt.Errorf("unknown memory backend: %s (supported: keyword, vector)", memCfg.Backend)
		}
	*/
}

// NewWorkingMemoryStrategyFromConfig creates a working memory strategy from configuration.
func NewWorkingMemoryStrategyFromConfig(cfg *config.ContextConfig, defaultModel string, llms map[string]model.LLM, defaultLLM model.LLM) (WorkingMemoryStrategy, error) {
	if cfg == nil {
		return NilWorkingMemory{}, nil
	}

	// Apply defaults
	cfg.SetDefaults()

	switch cfg.Strategy {
	case "none", "":
		return NilWorkingMemory{}, nil

	case "buffer_window":
		return NewBufferWindowStrategy(BufferWindowConfig{
			WindowSize: cfg.WindowSize,
		}), nil

	case "token_window":
		return NewTokenWindowStrategy(TokenWindowConfig{
			Budget:         cfg.Budget,
			PreserveRecent: cfg.PreserveRecent,
			Model:          defaultModel,
		})

	case "summary_buffer":
		// Get summarizer LLM
		summarizerLLMName := cfg.SummarizerLLM

		// Resolution logic for summarizer LLM:
		// 1. Configured SummarizerLLM
		// 2. Default LLM (passed from agent)
		var summarizerLLM model.LLM
		if summarizerLLMName != "" && llms != nil {
			var ok bool
			summarizerLLM, ok = llms[summarizerLLMName]
			if !ok {
				return nil, fmt.Errorf("summarizer LLM %q not found", summarizerLLMName)
			}
		} else {
			summarizerLLM = defaultLLM
		}

		// Detailed check:
		var summarizer Summarizer
		if summarizerLLM != nil {
			var err error
			summarizer, err = NewLLMSummarizer(LLMSummarizerConfig{
				LLM: summarizerLLM,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create summarizer: %w", err)
			}
		}

		return NewSummaryBufferStrategy(SummaryBufferConfig{
			Budget:      cfg.Budget,
			Threshold:   cfg.Threshold,
			Target:      cfg.Target,
			Model:       defaultModel,
			Summarizer:  summarizer,
			MinMessages: cfg.WindowSize, // Reuse window_size as min messages threshold
		})

	default:
		return nil, fmt.Errorf("unknown context strategy: %q", cfg.Strategy)
	}
}
