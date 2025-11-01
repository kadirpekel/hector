package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

// FieldInfo represents a config field definition
type FieldInfo struct {
	StructName string
	FieldName  string
	YAMLTag    string
	Type       string
	Location   string
	LineNumber int
}

// AccessInfo represents where a field is accessed
type AccessInfo struct {
	Package  string
	File     string
	Line     int
	Function string
}

// AnalysisResult contains the complete analysis
type AnalysisResult struct {
	AllFields        map[string]*FieldInfo     // key: StructName.FieldName
	ExternalAccesses map[string][]AccessInfo   // key: StructName.FieldName
	Timestamp        time.Time
}

func main() {
	fmt.Println("🔍 V2: External Access-Based Config Field Analysis")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	result := &AnalysisResult{
		AllFields:        make(map[string]*FieldInfo),
		ExternalAccesses: make(map[string][]AccessInfo),
		Timestamp:        time.Now(),
	}

	// Step 1: Find all config fields in pkg/config/types.go
	fmt.Println("📋 Step 1: Scanning config field definitions...")
	if err := scanConfigFields(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning config fields: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d config fields in %d structs\n\n", len(result.AllFields), countStructs(result.AllFields))

	// Step 2: Scan all packages OUTSIDE pkg/config for field accesses
	fmt.Println("🔎 Step 2: Scanning external packages for field accesses...")
	if err := scanExternalAccesses(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning external accesses: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Scanned packages outside pkg/config/\n\n")

	// Step 3: Generate report
	fmt.Println("📊 Step 3: Generating analysis report...")
	generateReport(result)
}

// scanConfigFields finds all config struct fields in pkg/config/types.go
func scanConfigFields(result *AnalysisResult) error {
	fset := token.NewFileSet()
	
	// Parse pkg/config/types.go
	file, err := parser.ParseFile(fset, "pkg/config/types.go", nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse types.go: %w", err)
	}

	// Walk the AST to find struct definitions
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for type declarations
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName := typeSpec.Name.Name

		// Scan all fields in the struct
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue // Embedded field
			}

			fieldName := field.Names[0].Name
			
			// Skip unexported fields
			if !ast.IsExported(fieldName) {
				continue
			}

			// Extract YAML tag
			yamlTag := ""
			if field.Tag != nil {
				tag := field.Tag.Value
				yamlTag = extractYAMLTag(tag)
			}

			// Get type
			fieldType := exprToString(field.Type)

			// Store field info
			key := fmt.Sprintf("%s.%s", structName, fieldName)
			result.AllFields[key] = &FieldInfo{
				StructName: structName,
				FieldName:  fieldName,
				YAMLTag:    yamlTag,
				Type:       fieldType,
				Location:   "pkg/config/types.go",
				LineNumber: fset.Position(field.Pos()).Line,
			}
		}

		return true
	})

	return nil
}

// scanExternalAccesses finds all field accesses OUTSIDE pkg/config
func scanExternalAccesses(result *AnalysisResult) error {
	// Load all packages
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  ".",
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	// Scan each package
	for _, pkg := range pkgs {
		// Skip config package itself
		if strings.Contains(pkg.PkgPath, "/pkg/config") {
			continue
		}

		// Skip test packages and tools
		if strings.HasSuffix(pkg.PkgPath, "_test") ||
			strings.Contains(pkg.PkgPath, "/tools/") ||
			strings.Contains(pkg.PkgPath, "/cmd/") {
			continue
		}

		// Scan files in this package
		for _, file := range pkg.Syntax {
			scanFileForAccesses(pkg, file, result)
		}
	}

	return nil
}

// scanFileForAccesses scans a single file for config field accesses
func scanFileForAccesses(pkg *packages.Package, file *ast.File, result *AnalysisResult) {
	filename := pkg.Fset.Position(file.Pos()).Filename
	
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for selector expressions (e.g., config.FieldName)
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Get the type of the selector's base expression
		typeInfo := pkg.TypesInfo.TypeOf(sel.X)
		if typeInfo == nil {
			return true
		}

		// Check if this is a config struct type
		typeName := typeInfo.String()
		
		// Extract struct name from type string
		// e.g., "*config.AgentConfig" -> "AgentConfig"
		//       "config.LLMProviderConfig" -> "LLMProviderConfig"
		structName := extractStructName(typeName)
		if structName == "" {
			return true
		}

		fieldName := sel.Sel.Name
		key := fmt.Sprintf("%s.%s", structName, fieldName)

		// Check if this is a known config field
		if _, exists := result.AllFields[key]; !exists {
			return true
		}

		// Record the access
		pos := pkg.Fset.Position(sel.Pos())
		access := AccessInfo{
			Package:  pkg.PkgPath,
			File:     filepath.Base(filename),
			Line:     pos.Line,
			Function: findEnclosingFunction(file, sel),
		}

		result.ExternalAccesses[key] = append(result.ExternalAccesses[key], access)

		return true
	})
}

// generateReport generates the final analysis report
func generateReport(result *AnalysisResult) {
	// Categorize fields
	unused := []string{}
	lightlyUsed := []string{}  // 1-2 accesses
	wellUsed := []string{}     // 3+ accesses

	for key := range result.AllFields {
		accesses := result.ExternalAccesses[key]
		accessCount := len(accesses)

		if accessCount == 0 {
			unused = append(unused, key)
		} else if accessCount <= 2 {
			lightlyUsed = append(lightlyUsed, key)
		} else {
			wellUsed = append(wellUsed, key)
		}
	}

	// Sort for consistent output
	sort.Strings(unused)
	sort.Strings(lightlyUsed)
	sort.Strings(wellUsed)

	// Calculate percentages
	total := len(result.AllFields)
	unusedPct := float64(len(unused)) / float64(total) * 100
	lightlyPct := float64(len(lightlyUsed)) / float64(total) * 100
	wellUsedPct := float64(len(wellUsed)) / float64(total) * 100

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("═", 70))
	fmt.Println("📊 EXTERNAL ACCESS ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Println()

	fmt.Printf("Total Fields:        %d\n", total)
	fmt.Printf("Unused (0 accesses): %d (%.1f%%)\n", len(unused), unusedPct)
	fmt.Printf("Lightly Used (1-2):  %d (%.1f%%)\n", len(lightlyUsed), lightlyPct)
	fmt.Printf("Well Used (3+):      %d (%.1f%%)\n", len(wellUsed), wellUsedPct)
	fmt.Println()

	// Status
	status := "⚠️  NEEDS WORK"
	if unusedPct < 5.0 {
		status = "✅ EXCELLENT"
	} else if unusedPct < 10.0 {
		status = "✅ HEALTHY"
	}
	fmt.Printf("Status: %s\n", status)
	fmt.Println()

	// Print unused fields
	if len(unused) > 0 {
		fmt.Println(strings.Repeat("═", 70))
		fmt.Printf("❌ UNUSED FIELDS (%d)\n", len(unused))
		fmt.Println(strings.Repeat("═", 70))
		fmt.Println()

		for _, key := range unused {
			field := result.AllFields[key]
			fmt.Printf("• %s.%s\n", field.StructName, field.FieldName)
			if field.YAMLTag != "" {
				fmt.Printf("  YAML: %s\n", field.YAMLTag)
			}
			fmt.Printf("  Type: %s\n", field.Type)
			fmt.Printf("  Location: %s:%d\n", field.Location, field.LineNumber)
			fmt.Printf("  ⚠️  NOT ACCESSED outside pkg/config\n")
			fmt.Println()
		}
	}

	// Print lightly used fields
	if len(lightlyUsed) > 0 {
		fmt.Println(strings.Repeat("═", 70))
		fmt.Printf("⚠️  LIGHTLY USED FIELDS (%d)\n", len(lightlyUsed))
		fmt.Println(strings.Repeat("═", 70))
		fmt.Println()

		for _, key := range lightlyUsed {
			field := result.AllFields[key]
			accesses := result.ExternalAccesses[key]
			
			fmt.Printf("• %s.%s (%d access(es))\n", field.StructName, field.FieldName, len(accesses))
			
			// Show where it's accessed
			for _, access := range accesses {
				fmt.Printf("  → %s (%s:%d)\n", access.Package, access.File, access.Line)
			}
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("═", 70))
	fmt.Println()
	fmt.Println("✅ Analysis complete!")
	fmt.Println()
	fmt.Println("💡 Key Insight:")
	fmt.Println("   Only fields accessed OUTSIDE pkg/config are counted.")
	fmt.Println("   Validation and SetDefaults are automatically excluded.")
	fmt.Println()
}

// Helper functions

func extractYAMLTag(tag string) string {
	// tag format: `yaml:"field_name,omitempty"`
	tag = strings.Trim(tag, "`")
	parts := strings.Split(tag, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "yaml:") {
			yamlPart := strings.TrimPrefix(part, "yaml:")
			yamlPart = strings.Trim(yamlPart, "\"")
			// Remove omitempty
			yamlPart = strings.Split(yamlPart, ",")[0]
			return yamlPart
		}
	}
	return ""
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprToString(t.Key), exprToString(t.Value))
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

func extractStructName(typeName string) string {
	// Remove pointer
	typeName = strings.TrimPrefix(typeName, "*")
	
	// Check if it's a config type
	if !strings.Contains(typeName, "config.") {
		return ""
	}
	
	// Extract struct name: "github.com/.../config.AgentConfig" -> "AgentConfig"
	parts := strings.Split(typeName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return ""
}

func findEnclosingFunction(file *ast.File, node ast.Node) string {
	var funcName string
	
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		
		// Check if we found a function declaration
		if fn, ok := n.(*ast.FuncDecl); ok {
			// Check if our node is inside this function
			if fn.Pos() <= node.Pos() && node.End() <= fn.End() {
				funcName = fn.Name.Name
				return false
			}
		}
		
		return true
	})
	
	return funcName
}

func countStructs(fields map[string]*FieldInfo) int {
	structs := make(map[string]bool)
	for _, field := range fields {
		structs[field.StructName] = true
	}
	return len(structs)
}

