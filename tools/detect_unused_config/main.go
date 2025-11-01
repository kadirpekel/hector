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
)

type ConfigField struct {
	StructName string
	FieldName  string
	Type       string
	YamlTag    string
	File       string
	Line       int
}

type FieldUsage struct {
	Field       ConfigField
	AccessCount int
	AccessFiles map[string]int
}

func main() {
	fmt.Fprintln(os.Stderr, "🔍 Analyzing configuration field usage...")
	fmt.Fprintln(os.Stderr, "")

	// Step 1: Find all config structs in pkg/config/types.go
	configFields, err := findConfigFields("pkg/config/types.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding config fields: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "✓ Found %d config fields in %d structs\n", len(configFields), countStructs(configFields))

	// Step 2: Search for field access patterns across entire codebase
	fmt.Fprintln(os.Stderr, "✓ Scanning codebase for field usage...")
	usage := analyzeFieldUsage(configFields)

	// Step 3: Report unused fields
	fmt.Fprintln(os.Stderr, "✓ Generating report...\n")
	reportUnusedFields(usage)
}

func countStructs(fields []ConfigField) int {
	structs := make(map[string]bool)
	for _, f := range fields {
		structs[f.StructName] = true
	}
	return len(structs)
}

func findConfigFields(configFile string) ([]ConfigField, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, configFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configFile, err)
	}

	var fields []ConfigField

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for struct declarations
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName := typeSpec.Name.Name

		// Only analyze config structs (end with "Config" or have "Config" in name)
		if !strings.Contains(structName, "Config") {
			return true
		}

		// Extract all fields
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				// Anonymous field, skip
				continue
			}

			fieldName := field.Names[0].Name

			// Skip unexported fields
			if !ast.IsExported(fieldName) {
				continue
			}

			// Extract yaml tag
			var yamlTag string
			if field.Tag != nil {
				tag := field.Tag.Value
				if strings.Contains(tag, "yaml:") {
					yamlTag = extractYamlTag(tag)
				}
			}

			// Get type as string
			typeStr := ""
			if field.Type != nil {
				typeStr = exprToString(field.Type)
			}

			fields = append(fields, ConfigField{
				StructName: structName,
				FieldName:  fieldName,
				Type:       typeStr,
				YamlTag:    yamlTag,
				File:       configFile,
				Line:       fset.Position(field.Pos()).Line,
			})
		}

		return true
	})

	return fields, nil
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
		return fmt.Sprintf("%T", t)
	}
}

func analyzeFieldUsage(fields []ConfigField) map[string]*FieldUsage {
	usage := make(map[string]*FieldUsage)

	for _, field := range fields {
		key := field.StructName + "." + field.FieldName
		usage[key] = &FieldUsage{
			Field:       field,
			AccessCount: 0,
			AccessFiles: make(map[string]int),
		}
	}

	// Walk entire codebase
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files, vendor, and config types file itself
		if !strings.HasSuffix(path, ".go") ||
			strings.Contains(path, "vendor/") ||
			strings.Contains(path, ".git/") ||
			strings.Contains(path, "tools/detect_unused_config") ||
			path == "pkg/config/types.go" {
			return nil
		}

		// Parse file and look for field access
		fset := token.NewFileSet()
		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // Skip files with parse errors
		}

		// Look for field accesses: obj.FieldName
		ast.Inspect(node, func(n ast.Node) bool {
			selector, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			fieldName := selector.Sel.Name

			// Check if this matches any of our config fields
			for key, u := range usage {
				if strings.HasSuffix(key, "."+fieldName) {
					u.AccessCount++
					u.AccessFiles[path]++
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error walking codebase: %v\n", err)
	}

	return usage
}

func reportUnusedFields(usage map[string]*FieldUsage) {
	fmt.Println("# Configuration Field Usage Report")
	fmt.Println("")
	fmt.Println("Generated: " + fmt.Sprint(os.Getenv("USER")) + " @ " + fmt.Sprint(os.Getenv("HOSTNAME")))
	fmt.Println("")

	// Categorize fields
	unused := []*FieldUsage{}
	lightlyUsed := []*FieldUsage{}
	used := []*FieldUsage{}

	for _, u := range usage {
		if u.AccessCount == 0 {
			unused = append(unused, u)
		} else if u.AccessCount < 3 {
			lightlyUsed = append(lightlyUsed, u)
		} else {
			used = append(used, u)
		}
	}

	// Sort by struct name, then field name
	sortUsage := func(slice []*FieldUsage) {
		sort.Slice(slice, func(i, j int) bool {
			if slice[i].Field.StructName != slice[j].Field.StructName {
				return slice[i].Field.StructName < slice[j].Field.StructName
			}
			return slice[i].Field.FieldName < slice[j].Field.FieldName
		})
	}

	sortUsage(unused)
	sortUsage(lightlyUsed)
	sortUsage(used)

	// Summary
	fmt.Println("## 📊 Summary")
	fmt.Println("")
	fmt.Printf("| Metric | Count | Percentage |\n")
	fmt.Printf("|--------|-------|------------|\n")
	fmt.Printf("| **Total fields** | %d | 100%% |\n", len(usage))
	fmt.Printf("| **Unused fields** | %d | %.1f%% |\n", len(unused), float64(len(unused))/float64(len(usage))*100)
	fmt.Printf("| **Lightly used (1-2 accesses)** | %d | %.1f%% |\n", len(lightlyUsed), float64(len(lightlyUsed))/float64(len(usage))*100)
	fmt.Printf("| **Well used (3+ accesses)** | %d | %.1f%% |\n", len(used), float64(len(used))/float64(len(usage))*100)
	fmt.Println("")

	// Unused fields
	if len(unused) > 0 {
		fmt.Println("## ❌ Unused Fields")
		fmt.Println("")
		fmt.Printf("Found **%d unused fields** that should be considered for removal:\n", len(unused))
		fmt.Println("")

		currentStruct := ""
		for _, u := range unused {
			if u.Field.StructName != currentStruct {
				if currentStruct != "" {
					fmt.Println("")
				}
				currentStruct = u.Field.StructName
				fmt.Printf("### %s\n\n", currentStruct)
			}

			fmt.Printf("- **`%s`**\n", u.Field.FieldName)
			if u.Field.YamlTag != "" {
				fmt.Printf("  - YAML: `%s`\n", u.Field.YamlTag)
			}
			fmt.Printf("  - Type: `%s`\n", u.Field.Type)
			fmt.Printf("  - Location: `%s:%d`\n", u.Field.File, u.Field.Line)
			fmt.Printf("  - Accesses: **0** ⚠️\n")
		}
		fmt.Println("")
	} else {
		fmt.Println("## ✅ No Unused Fields")
		fmt.Println("")
		fmt.Println("All configuration fields are being used! 🎉")
		fmt.Println("")
	}

	// Lightly used fields
	if len(lightlyUsed) > 0 {
		fmt.Println("## ⚠️  Lightly Used Fields")
		fmt.Println("")
		fmt.Printf("Found **%d lightly used fields** (1-2 accesses) - verify they're necessary:\n", len(lightlyUsed))
		fmt.Println("")

		currentStruct := ""
		for _, u := range lightlyUsed {
			if u.Field.StructName != currentStruct {
				if currentStruct != "" {
					fmt.Println("")
				}
				currentStruct = u.Field.StructName
				fmt.Printf("### %s\n\n", currentStruct)
			}

			fmt.Printf("- **`%s`** - %d access(es) in %d file(s)\n", u.Field.FieldName, u.AccessCount, len(u.AccessFiles))
			if u.Field.YamlTag != "" {
				fmt.Printf("  - YAML: `%s`\n", u.Field.YamlTag)
			}

			// Show which files access it
			if len(u.AccessFiles) > 0 {
				fmt.Printf("  - Used in: ")
				files := make([]string, 0, len(u.AccessFiles))
				for f := range u.AccessFiles {
					files = append(files, f)
				}
				sort.Strings(files)
				for i, f := range files {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("`%s`", f)
				}
				fmt.Println()
			}
		}
		fmt.Println("")
	}

	// Well used fields (top 20)
	if len(used) > 0 {
		fmt.Println("## ✅ Well Used Fields (Top 20)")
		fmt.Println("")
		fmt.Println("Fields with 3+ accesses across the codebase:")
		fmt.Println("")

		// Sort by access count descending
		sort.Slice(used, func(i, j int) bool {
			return used[i].AccessCount > used[j].AccessCount
		})

		limit := 20
		if len(used) < limit {
			limit = len(used)
		}

		for i := 0; i < limit; i++ {
			u := used[i]
			fmt.Printf("- **`%s.%s`** - %d accesses in %d files\n",
				u.Field.StructName, u.Field.FieldName, u.AccessCount, len(u.AccessFiles))
		}

		if len(used) > limit {
			fmt.Printf("\n*... and %d more well-used fields*\n", len(used)-limit)
		}
		fmt.Println("")
	}

	// Recommendations
	fmt.Println("## 💡 Recommendations")
	fmt.Println("")

	if len(unused) > 0 {
		fmt.Printf("1. **Remove %d unused fields** to reduce code complexity\n", len(unused))
		fmt.Println("2. Update documentation to remove references to deleted fields")
		fmt.Println("3. Update example configs if they use these fields")
	}

	if len(lightlyUsed) > 0 {
		fmt.Printf("4. **Review %d lightly-used fields** - are they truly needed?\n", len(lightlyUsed))
		fmt.Println("5. Consider if lightly-used fields should be removed or better utilized")
	}

	if len(unused) == 0 && len(lightlyUsed) == 0 {
		fmt.Println("✅ Configuration is healthy! All fields are well-utilized.")
	}

	fmt.Println("")
	fmt.Println("---")
	fmt.Println("")
	fmt.Println("*Run this report regularly to keep configuration clean and maintainable.*")
}

func extractYamlTag(tag string) string {
	// Extract yaml:"field_name" from tag like `yaml:"field_name,omitempty"`
	start := strings.Index(tag, "yaml:\"")
	if start == -1 {
		return ""
	}
	start += 6
	end := strings.Index(tag[start:], "\"")
	if end == -1 {
		return ""
	}
	result := tag[start : start+end]

	// Remove ,omitempty and other modifiers
	if commaIdx := strings.Index(result, ","); commaIdx != -1 {
		result = result[:commaIdx]
	}

	return result
}

