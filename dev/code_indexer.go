// Package dev provides code indexing for semantic search
package dev

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// CODE INDEXER - Parse Go files and extract symbols
// ============================================================================

// CodeIndexer parses Go files and extracts functions, types, interfaces
type CodeIndexer struct {
	ProjectRoot string
	Verbose     bool
}

// NewCodeIndexer creates a new code indexer
func NewCodeIndexer(projectRoot string) *CodeIndexer {
	return &CodeIndexer{
		ProjectRoot: projectRoot,
		Verbose:     false,
	}
}

// CodeSymbol represents a Go code symbol (function, type, interface, etc.)
type CodeSymbol struct {
	Type      SymbolType `json:"type"`               // function, type, interface, struct, const, var
	Package   string     `json:"package"`            // Package name
	Name      string     `json:"name"`               // Symbol name
	Signature string     `json:"signature"`          // Full signature
	Doc       string     `json:"doc"`                // Documentation comment
	File      string     `json:"file"`               // Source file path
	Line      int        `json:"line"`               // Line number
	Body      string     `json:"body"`               // Function/method body
	Fields    []string   `json:"fields,omitempty"`   // Struct fields
	Methods   []string   `json:"methods,omitempty"`  // Methods (for interfaces)
	Receiver  string     `json:"receiver,omitempty"` // Receiver type (for methods)
}

// SymbolType represents the type of code symbol
type SymbolType string

const (
	SymbolTypeFunction  SymbolType = "function"
	SymbolTypeMethod    SymbolType = "method"
	SymbolTypeType      SymbolType = "type"
	SymbolTypeStruct    SymbolType = "struct"
	SymbolTypeInterface SymbolType = "interface"
	SymbolTypeConst     SymbolType = "const"
	SymbolTypeVar       SymbolType = "var"
)

// IndexResult contains all indexed symbols
type IndexResult struct {
	Symbols      []CodeSymbol       `json:"symbols"`
	FileCount    int                `json:"file_count"`
	SymbolCounts map[SymbolType]int `json:"symbol_counts"`
}

// Index indexes all Go files in specified directories
func (idx *CodeIndexer) Index(directories []string) (*IndexResult, error) {
	result := &IndexResult{
		Symbols:      make([]CodeSymbol, 0),
		SymbolCounts: make(map[SymbolType]int),
	}

	for _, dir := range directories {
		fullPath := filepath.Join(idx.ProjectRoot, dir)

		if idx.Verbose {
			fmt.Printf("Indexing directory: %s\n", fullPath)
		}

		err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-Go files
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			// Skip test files
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			// Parse file
			symbols, err := idx.parseFile(path)
			if err != nil {
				if idx.Verbose {
					fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
				}
				return nil // Continue on error
			}

			result.Symbols = append(result.Symbols, symbols...)
			result.FileCount++

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}

	// Count symbols by type
	for _, symbol := range result.Symbols {
		result.SymbolCounts[symbol.Type]++
	}

	return result, nil
}

// parseFile parses a single Go file and extracts symbols
func (idx *CodeIndexer) parseFile(filepath string) ([]CodeSymbol, error) {
	// Parse file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	symbols := make([]CodeSymbol, 0)
	packageName := node.Name.Name

	// Extract symbols
	ast.Inspect(node, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			symbol := idx.extractFunction(decl, packageName, filepath, fset)
			symbols = append(symbols, symbol)

		case *ast.GenDecl:
			extracted := idx.extractGenDecl(decl, packageName, filepath, fset)
			symbols = append(symbols, extracted...)
		}
		return true
	})

	return symbols, nil
}

// extractFunction extracts a function or method declaration
func (idx *CodeIndexer) extractFunction(decl *ast.FuncDecl, pkg, file string, fset *token.FileSet) CodeSymbol {
	symbol := CodeSymbol{
		Package: pkg,
		Name:    decl.Name.Name,
		File:    file,
		Line:    fset.Position(decl.Pos()).Line,
	}

	// Extract documentation
	if decl.Doc != nil {
		symbol.Doc = decl.Doc.Text()
	}

	// Determine if function or method
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		symbol.Type = SymbolTypeMethod
		// Extract receiver type
		if recvType, ok := decl.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := recvType.X.(*ast.Ident); ok {
				symbol.Receiver = ident.Name
			}
		} else if ident, ok := decl.Recv.List[0].Type.(*ast.Ident); ok {
			symbol.Receiver = ident.Name
		}
	} else {
		symbol.Type = SymbolTypeFunction
	}

	// Build signature
	signature := idx.buildFunctionSignature(decl)
	symbol.Signature = signature

	// Extract body (first 500 chars)
	if decl.Body != nil {
		bodyStart := fset.Position(decl.Body.Pos()).Offset
		bodyEnd := fset.Position(decl.Body.End()).Offset

		// Read file content
		content, err := os.ReadFile(file)
		if err == nil && bodyStart < len(content) && bodyEnd <= len(content) {
			body := string(content[bodyStart:bodyEnd])
			if len(body) > 500 {
				body = body[:500] + "..."
			}
			symbol.Body = body
		}
	}

	return symbol
}

// buildFunctionSignature builds a function signature string
func (idx *CodeIndexer) buildFunctionSignature(decl *ast.FuncDecl) string {
	var sig strings.Builder

	sig.WriteString("func ")

	// Add receiver for methods
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		sig.WriteString("(")
		// Simplified receiver representation
		if st, ok := decl.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := st.X.(*ast.Ident); ok {
				sig.WriteString("*" + ident.Name)
			}
		} else if ident, ok := decl.Recv.List[0].Type.(*ast.Ident); ok {
			sig.WriteString(ident.Name)
		}
		sig.WriteString(") ")
	}

	sig.WriteString(decl.Name.Name)
	sig.WriteString("(")

	// Add parameters
	if decl.Type.Params != nil {
		for i, param := range decl.Type.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			// Simplified param representation
			if len(param.Names) > 0 {
				sig.WriteString(param.Names[0].Name)
				sig.WriteString(" ")
			}
			sig.WriteString(idx.exprToString(param.Type))
		}
	}

	sig.WriteString(")")

	// Add return type
	if decl.Type.Results != nil && len(decl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(decl.Type.Results.List) == 1 && len(decl.Type.Results.List[0].Names) == 0 {
			sig.WriteString(idx.exprToString(decl.Type.Results.List[0].Type))
		} else {
			sig.WriteString("(")
			for i, result := range decl.Type.Results.List {
				if i > 0 {
					sig.WriteString(", ")
				}
				sig.WriteString(idx.exprToString(result.Type))
			}
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// extractGenDecl extracts type, const, var declarations
func (idx *CodeIndexer) extractGenDecl(decl *ast.GenDecl, pkg, file string, fset *token.FileSet) []CodeSymbol {
	symbols := make([]CodeSymbol, 0)

	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			symbol := idx.extractTypeSpec(s, decl, pkg, file, fset)
			symbols = append(symbols, symbol)

		case *ast.ValueSpec:
			extracted := idx.extractValueSpec(s, decl, pkg, file, fset)
			symbols = append(symbols, extracted...)
		}
	}

	return symbols
}

// extractTypeSpec extracts type declarations (struct, interface, etc.)
func (idx *CodeIndexer) extractTypeSpec(spec *ast.TypeSpec, decl *ast.GenDecl, pkg, file string, fset *token.FileSet) CodeSymbol {
	symbol := CodeSymbol{
		Package: pkg,
		Name:    spec.Name.Name,
		File:    file,
		Line:    fset.Position(spec.Pos()).Line,
	}

	// Extract documentation
	if decl.Doc != nil {
		symbol.Doc = decl.Doc.Text()
	}

	// Determine type
	switch t := spec.Type.(type) {
	case *ast.StructType:
		symbol.Type = SymbolTypeStruct
		symbol.Signature = fmt.Sprintf("type %s struct", spec.Name.Name)

		// Extract fields
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				for _, name := range field.Names {
					fieldStr := fmt.Sprintf("%s %s", name.Name, idx.exprToString(field.Type))
					symbol.Fields = append(symbol.Fields, fieldStr)
				}
			}
		}

	case *ast.InterfaceType:
		symbol.Type = SymbolTypeInterface
		symbol.Signature = fmt.Sprintf("type %s interface", spec.Name.Name)

		// Extract methods
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				if len(method.Names) > 0 {
					methodStr := fmt.Sprintf("%s%s", method.Names[0].Name, idx.exprToString(method.Type))
					symbol.Methods = append(symbol.Methods, methodStr)
				}
			}
		}

	default:
		symbol.Type = SymbolTypeType
		symbol.Signature = fmt.Sprintf("type %s %s", spec.Name.Name, idx.exprToString(spec.Type))
	}

	return symbol
}

// extractValueSpec extracts const/var declarations
func (idx *CodeIndexer) extractValueSpec(spec *ast.ValueSpec, decl *ast.GenDecl, pkg, file string, fset *token.FileSet) []CodeSymbol {
	symbols := make([]CodeSymbol, 0)

	symbolType := SymbolTypeVar
	if decl.Tok == token.CONST {
		symbolType = SymbolTypeConst
	}

	for _, name := range spec.Names {
		symbol := CodeSymbol{
			Type:    symbolType,
			Package: pkg,
			Name:    name.Name,
			File:    file,
			Line:    fset.Position(name.Pos()).Line,
		}

		// Extract documentation
		if decl.Doc != nil {
			symbol.Doc = decl.Doc.Text()
		}

		// Build signature
		if spec.Type != nil {
			symbol.Signature = fmt.Sprintf("%s %s %s", decl.Tok.String(), name.Name, idx.exprToString(spec.Type))
		} else if len(spec.Values) > 0 {
			symbol.Signature = fmt.Sprintf("%s %s = ...", decl.Tok.String(), name.Name)
		}

		symbols = append(symbols, symbol)
	}

	return symbols
}

// exprToString converts an expression to a string representation
func (idx *CodeIndexer) exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + idx.exprToString(e.X)
	case *ast.ArrayType:
		return "[]" + idx.exprToString(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", idx.exprToString(e.Key), idx.exprToString(e.Value))
	case *ast.SelectorExpr:
		return idx.exprToString(e.X) + "." + e.Sel.Name
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + idx.exprToString(e.Value)
	default:
		return "..."
	}
}

// ============================================================================
// FORMATTING & OUTPUT
// ============================================================================

// FormatSummary returns a human-readable summary of indexing results
func (result *IndexResult) FormatSummary() string {
	var out strings.Builder

	out.WriteString("\n╔═══════════════════════════════════════════════════════════╗\n")
	out.WriteString("║              CODE INDEXING RESULTS                        ║\n")
	out.WriteString("╚═══════════════════════════════════════════════════════════╝\n\n")

	out.WriteString(fmt.Sprintf("Files Indexed: %d\n", result.FileCount))
	out.WriteString(fmt.Sprintf("Total Symbols: %d\n\n", len(result.Symbols)))

	out.WriteString("Symbols by Type:\n")
	out.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for _, symType := range []SymbolType{
		SymbolTypeFunction,
		SymbolTypeMethod,
		SymbolTypeStruct,
		SymbolTypeInterface,
		SymbolTypeType,
		SymbolTypeConst,
		SymbolTypeVar,
	} {
		if count, ok := result.SymbolCounts[symType]; ok && count > 0 {
			out.WriteString(fmt.Sprintf("  %-12s: %d\n", symType, count))
		}
	}

	out.WriteString("\n")
	return out.String()
}
