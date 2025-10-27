package metadata

// MetadataExtractor defines the interface for extracting metadata from source code
type MetadataExtractor interface {
	// Name returns the extractor name
	Name() string

	// CanExtract determines if this extractor can handle the given language
	CanExtract(language string) bool

	// Extract extracts metadata from source code
	Extract(content string, filePath string) (*Metadata, error)
}

// Metadata contains extracted code structure information
type Metadata struct {
	Functions []FunctionInfo            `json:"functions,omitempty"`
	Types     []TypeInfo                `json:"types,omitempty"`
	Imports   []string                  `json:"imports,omitempty"`
	Symbols   map[string]interface{}    `json:"symbols,omitempty"`
	Custom    map[string]interface{}    `json:"custom,omitempty"`
}

// FunctionInfo contains information about a function
type FunctionInfo struct {
	Name       string `json:"name"`
	Signature  string `json:"signature,omitempty"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Receiver   string `json:"receiver,omitempty"`   // For methods
	IsExported bool   `json:"is_exported,omitempty"`
	DocComment string `json:"doc_comment,omitempty"`
}

// TypeInfo contains information about a type (struct, interface, etc.)
type TypeInfo struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // "struct", "interface", "alias", etc.
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
	IsExported bool     `json:"is_exported,omitempty"`
	DocComment string   `json:"doc_comment,omitempty"`
}

// ExtractorRegistry manages metadata extractors
type ExtractorRegistry struct {
	extractors map[string]MetadataExtractor
}

// NewExtractorRegistry creates a new metadata extractor registry
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: make(map[string]MetadataExtractor),
	}
}

// Register adds a metadata extractor for specific languages
func (r *ExtractorRegistry) Register(extractor MetadataExtractor) {
	r.extractors[extractor.Name()] = extractor
}

// ExtractMetadata tries to extract metadata using the appropriate extractor
func (r *ExtractorRegistry) ExtractMetadata(language string, content string, filePath string) (*Metadata, error) {
	for _, extractor := range r.extractors {
		if extractor.CanExtract(language) {
			return extractor.Extract(content, filePath)
		}
	}

	// No extractor found - return empty metadata
	return &Metadata{
		Functions: []FunctionInfo{},
		Types:     []TypeInfo{},
		Imports:   []string{},
		Symbols:   make(map[string]interface{}),
		Custom:    make(map[string]interface{}),
	}, nil
}

// GetExtractors returns all registered extractors
func (r *ExtractorRegistry) GetExtractors() []MetadataExtractor {
	extractors := make([]MetadataExtractor, 0, len(r.extractors))
	for _, ext := range r.extractors {
		extractors = append(extractors, ext)
	}
	return extractors
}
