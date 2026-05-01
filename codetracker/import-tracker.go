// this tracker is to track what are the modules imported in the file
// this tracks whether imported modules are used in the code
// tracks what are in code modules and external modules .

package codetracker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type ImportMeta struct {
	Path      string   `json:"path"`
	Alias     string   `json:"alias,omitempty"`
	LocalName string   `json:"local_name"`
	IsUsed    bool     `json:"is_used"`
	IsStdLib  bool     `json:"is_stdlib"`
	Comments  []string `json:"comments,omitempty"`
}

func ImportTracker(filePath string) ([]ImportMeta, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var imports []ImportMeta

	// collect all imports
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		meta := ImportMeta{
			Path:     path,
			IsStdLib: isStdLib(path),
		}

		if imp.Name != nil {
			meta.Alias = imp.Name.Name
		}

		meta.LocalName = localName(path, meta.Alias)

		if imp.Doc != nil {
			for _, c := range imp.Doc.List {
				meta.Comments = append(meta.Comments, c.Text)
			}
		}
		if imp.Comment != nil {
			for _, c := range imp.Comment.List {
				meta.Comments = append(meta.Comments, c.Text)
			}
		}

		imports = append(imports, meta)
	}

	// find which imports are actually used by scanning SelectorExpr nodes
	usedNames := collectUsedPackageNames(node)

	for i, imp := range imports {
		// blank imports are intentional side-effect imports — always mark used
		if imp.Alias == "_" {
			imports[i].IsUsed = true
			continue
		}
		// dot imports bring all names into scope — we can't easily track individual refs
		if imp.Alias == "." {
			imports[i].IsUsed = true
			continue
		}
		if usedNames[imp.LocalName] {
			imports[i].IsUsed = true
		}
	}

	return imports, nil
}

// isStdLib returns true when the first path segment has no dot (e.g. "fmt", "net/http")
func isStdLib(path string) bool {
	firstSegment := strings.SplitN(path, "/", 2)[0]
	return !strings.Contains(firstSegment, ".")
}

// localName returns the identifier used to reference the package in source code
func localName(path, alias string) string {
	if alias != "" && alias != "_" && alias != "." {
		return alias
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// collectUsedPackageNames walks the AST and returns every package name that
// appears as the left-hand side of a selector expression (e.g. fmt.Println → "fmt")
func collectUsedPackageNames(node *ast.File) map[string]bool {
	used := make(map[string]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok {
			used[ident.Name] = true
		}
		return true
	})
	return used
}
