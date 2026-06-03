// Command postprocess applies transformations to generated.go to fix conflicts
// introduced by oapi-codegen that cannot be resolved via configuration alone.
//
// Transformations applied:
//  1. Rename symbols that conflict with the public API of this package
//     (ClientOption, NewClient, WithHTTPClient, WithBaseURL, PrometheusRule,
//     and the PrometheusResultType "string" enum value, which oapi-codegen
//     names String and would clash with the hand-written String helper).
//     oapi-codegen does not provide a configuration option to rename these.
//     Selector members (e.g. url.URL.String()) are left untouched so the
//     rename never rewrites a method or field access on a different type.
//  2. Remove deprecated fields from PrometheusAlertRule (Description, Summary,
//     and the deprecated KeepFiringFor) since these have been replaced by
//     annotations and the snake_case keep_firing_for field respectively.
//
// Usage: go run ./tools/postprocess generated.go
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

// symbolRenames maps generated symbol names to their replacements. These
// renames prevent the generated code from exporting symbols that clash with
// the hand-written public API.
var symbolRenames = map[string]string{
	"ClientOption":   "generatedClientOption",
	"NewClient":      "newGeneratedClient",
	"WithHTTPClient": "withGeneratedHTTPClient",
	"WithBaseURL":    "withGeneratedApiUrl",
	"PrometheusRule": "generatedPrometheusRule",
	// The PrometheusResultType "string" value is generated as the const String,
	// which clashes with the hand-written String helper in util.go.
	"String": "PrometheusResultTypeString",
}

// deprecatedFields lists fields to remove from specific structs. Each entry
// maps a struct type name to a set of field names whose doc comments contain
// "Deprecated". Only fields with deprecated comments are actually removed, so
// if a field is listed here but the spec drops the deprecation marker in the
// future, it will be left alone.
var deprecatedFields = map[string]map[string]bool{
	"PrometheusAlertRule": {
		"Description":   true,
		"Summary":       true,
		"KeepFiringFor": true,
	},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <file.go>\n", os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	renameSymbols(file)
	removeDeprecatedFields(file, fset)
	// Note: we intentionally do NOT remove duplicate struct fields. If
	// oapi-codegen produces duplicates, the build should fail loudly so the
	// OpenAPI spec bug is surfaced rather than silently papered over.

	out, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
	cfg := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(out, fset, file); err != nil {
		fmt.Fprintf(os.Stderr, "print error: %v\n", err)
		os.Exit(1)
	}
	if err := out.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "close error: %v\n", err)
		os.Exit(1)
	}
}

// renameSymbols replaces occurrences of conflicting identifiers throughout the
// AST. Selector members (the Sel in expressions like url.URL.String()) are left
// untouched, so a rename never rewrites a method or field access on a different
// type that happens to share a name with a generated symbol.
func renameSymbols(file *ast.File) {
	ast.Walk(symbolRenamer{}, file)
}

// symbolRenamer is an ast.Visitor that applies symbolRenames to identifiers,
// skipping the selector member of selector expressions.
type symbolRenamer struct{}

func (r symbolRenamer) Visit(n ast.Node) ast.Visitor {
	switch node := n.(type) {
	case *ast.SelectorExpr:
		// Rename within the X expression, but never the selector member: it
		// resolves to a method or field on another type (for example the
		// standard library's url.URL.String()).
		ast.Walk(r, node.X)
		return nil
	case *ast.Ident:
		if replacement, found := symbolRenames[node.Name]; found {
			node.Name = replacement
		}
		return nil
	}
	return r
}

// removeDeprecatedFields removes fields listed in the deprecatedFields map
// from their respective structs, but only if the field's doc comment actually
// contains "Deprecated".
func removeDeprecatedFields(file *ast.File, fset *token.FileSet) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			fields, ok := deprecatedFields[typeSpec.Name.Name]
			if !ok {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				continue
			}

			var toRemove []int
			for i, field := range st.Fields.List {
				for _, name := range field.Names {
					if !fields[name.Name] {
						continue
					}
					if isDeprecated(field, file, fset) {
						toRemove = append(toRemove, i)
						fmt.Fprintf(os.Stderr, "postprocess: removing deprecated field %s.%s at %s\n",
							typeSpec.Name.Name, name.Name, fset.Position(field.Pos()))
					}
				}
			}

			if len(toRemove) > 0 {
				removeFieldsByIndex(st, toRemove, file, fset)
			}
		}
	}
}

// isDeprecated checks whether a field has a doc comment or an associated
// comment group containing "Deprecated".
func isDeprecated(field *ast.Field, file *ast.File, fset *token.FileSet) bool {
	if field.Doc != nil {
		for _, c := range field.Doc.List {
			if strings.Contains(c.Text, "Deprecated") {
				return true
			}
		}
	}
	// Also check free-floating comment groups that appear right before the field.
	fieldLine := fset.Position(field.Pos()).Line
	for _, cg := range file.Comments {
		cgEndLine := fset.Position(cg.End()).Line
		if cgEndLine == fieldLine-1 || cgEndLine == fieldLine {
			for _, c := range cg.List {
				if strings.Contains(c.Text, "Deprecated") {
					return true
				}
			}
		}
	}
	return false
}

// removeFieldsByIndex removes fields at the given indices from a struct,
// including any associated doc/deprecated comment groups.
func removeFieldsByIndex(st *ast.StructType, indices []int, file *ast.File, fset *token.FileSet) {
	remove := map[int]bool{}
	for _, idx := range indices {
		remove[idx] = true
	}

	var removedPositions []token.Pos
	for idx := range remove {
		removedPositions = append(removedPositions, st.Fields.List[idx].Pos())
	}

	filtered := make([]*ast.Field, 0, len(st.Fields.List)-len(remove))
	for i, f := range st.Fields.List {
		if !remove[i] {
			filtered = append(filtered, f)
		}
	}
	st.Fields.List = filtered

	// Remove comment groups associated with removed fields.
	var filteredComments []*ast.CommentGroup
	for _, cg := range file.Comments {
		shouldRemove := false
		for _, pos := range removedPositions {
			cgEnd := fset.Position(cg.End())
			fieldStart := fset.Position(pos)
			if cgEnd.Line == fieldStart.Line || cgEnd.Line == fieldStart.Line-1 {
				for _, c := range cg.List {
					if strings.Contains(c.Text, "Deprecated") {
						shouldRemove = true
						break
					}
				}
			}
		}
		if !shouldRemove {
			filteredComments = append(filteredComments, cg)
		}
	}
	file.Comments = filteredComments
}
