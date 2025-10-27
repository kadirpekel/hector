package metadata

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoExtractor extracts metadata from Go source files using AST parsing
type GoExtractor struct{}

// NewGoExtractor creates a new Go metadata extractor
func NewGoExtractor() *GoExtractor {
	return &GoExtractor{}
}

// Name returns the extractor name
func (ge *GoExtractor) Name() string {
	return "GoExtractor"
}

// CanExtract checks if this extractor can handle the language
func (ge *GoExtractor) CanExtract(language string) bool {
	return language == "go"
}

// Extract parses Go source code and extracts metadata
func (ge *GoExtractor) Extract(content string, filePath string) (*Metadata, error) {
	fset := token.NewFileSet()

	// Parse the Go source file
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		// Return empty metadata on parse error
		return &Metadata{
			Functions: []FunctionInfo{},
			Types:     []TypeInfo{},
			Imports:   []string{},
			Symbols:   make(map[string]interface{}),
			Custom:    make(map[string]interface{}),
		}, nil
	}

	metadata := &Metadata{
		Functions: make([]FunctionInfo, 0),
		Types:     make([]TypeInfo, 0),
		Imports:   make([]string, 0),
		Symbols:   make(map[string]interface{}),
		Custom:    make(map[string]interface{}),
	}

	// Extract imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		metadata.Imports = append(metadata.Imports, importPath)
	}

	// Walk the AST and extract declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			ge.extractFunction(node, fset, metadata)
		case *ast.GenDecl:
			ge.extractGenDecl(node, fset, metadata)
		}
		return true
	})

	return metadata, nil
}

// extractFunction extracts information about a function or method
func (ge *GoExtractor) extractFunction(funcDecl *ast.FuncDecl, fset *token.FileSet, metadata *Metadata) {
	startPos := fset.Position(funcDecl.Pos())
	endPos := fset.Position(funcDecl.End())

	funcInfo := FunctionInfo{
		Name:       funcDecl.Name.Name,
		StartLine:  startPos.Line,
		EndLine:    endPos.Line,
		IsExported: ast.IsExported(funcDecl.Name.Name),
	}

	// Extract receiver for methods
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recvType := funcDecl.Recv.List[0].Type
		funcInfo.Receiver = ge.formatType(recvType)
	}

	// Extract function signature
	funcInfo.Signature = ge.formatFuncSignature(funcDecl)

	// Extract doc comment
	if funcDecl.Doc != nil {
		funcInfo.DocComment = funcDecl.Doc.Text()
	}

	metadata.Functions = append(metadata.Functions, funcInfo)
}

// extractGenDecl extracts type declarations, constants, and variables
func (ge *GoExtractor) extractGenDecl(genDecl *ast.GenDecl, fset *token.FileSet, metadata *Metadata) {
	if genDecl.Tok != token.TYPE {
		return
	}

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		startPos := fset.Position(typeSpec.Pos())
		endPos := fset.Position(typeSpec.End())

		typeInfo := TypeInfo{
			Name:       typeSpec.Name.Name,
			StartLine:  startPos.Line,
			EndLine:    endPos.Line,
			IsExported: ast.IsExported(typeSpec.Name.Name),
			Fields:     make([]string, 0),
			Methods:    make([]string, 0),
		}

		// Extract doc comment
		if genDecl.Doc != nil {
			typeInfo.DocComment = genDecl.Doc.Text()
		}

		// Determine type kind and extract fields
		switch t := typeSpec.Type.(type) {
		case *ast.StructType:
			typeInfo.Kind = "struct"
			if t.Fields != nil {
				for _, field := range t.Fields.List {
					for _, name := range field.Names {
						typeInfo.Fields = append(typeInfo.Fields, name.Name)
					}
				}
			}
		case *ast.InterfaceType:
			typeInfo.Kind = "interface"
			if t.Methods != nil {
				for _, method := range t.Methods.List {
					for _, name := range method.Names {
						typeInfo.Methods = append(typeInfo.Methods, name.Name)
					}
				}
			}
		default:
			typeInfo.Kind = "alias"
		}

		metadata.Types = append(metadata.Types, typeInfo)
	}
}

// formatFuncSignature creates a human-readable function signature
func (ge *GoExtractor) formatFuncSignature(funcDecl *ast.FuncDecl) string {
	var sb strings.Builder

	sb.WriteString("func ")

	// Add receiver if present
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		sb.WriteString("(")
		sb.WriteString(ge.formatType(funcDecl.Recv.List[0].Type))
		sb.WriteString(") ")
	}

	sb.WriteString(funcDecl.Name.Name)
	sb.WriteString("(")

	// Add parameters
	if funcDecl.Type.Params != nil {
		for i, param := range funcDecl.Type.Params.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			for j, name := range param.Names {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(name.Name)
			}
			if len(param.Names) > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(ge.formatType(param.Type))
		}
	}

	sb.WriteString(")")

	// Add return types
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(funcDecl.Type.Results.List) > 1 {
			sb.WriteString("(")
		}
		for i, result := range funcDecl.Type.Results.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(ge.formatType(result.Type))
		}
		if len(funcDecl.Type.Results.List) > 1 {
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// formatType converts an AST type expression to a string
func (ge *GoExtractor) formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + ge.formatType(t.X)
	case *ast.ArrayType:
		return "[]" + ge.formatType(t.Elt)
	case *ast.SelectorExpr:
		return ge.formatType(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + ge.formatType(t.Key) + "]" + ge.formatType(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}
