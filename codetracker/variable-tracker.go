package codetracker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type VariableMeta struct {
	Name     string   `json:"name"`
	Type     string   `json:"type,omitempty"`
	Value    any      `json:"value,omitempty"`
	Line     int      `json:"line"`
	Comments []string `json:"comments,omitempty"`
}

// TrackVariables parses a file and returns metadata about variables found.
func TrackVariables(filePath string) ([]VariableMeta, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var variables []VariableMeta

	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range genDecl.Specs {
			vset, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range vset.Names {
				meta := VariableMeta{
					Name: name.Name,
					Line: fset.Position(name.Pos()).Line,
				}

				if vset.Type != nil {
					meta.Type = fmt.Sprintf("%s", vset.Type)
				}

				if len(vset.Values) > i {
					meta.Value = fmt.Sprintf("%+v", vset.Values[i])
				}

				// doc comment above the var declaration (e.g. // this variable ...)
				if genDecl.Doc != nil {
					for _, c := range genDecl.Doc.List {
						meta.Comments = append(meta.Comments, c.Text)
					}
				}

				// per-spec doc comment (inside grouped var blocks)
				if vset.Doc != nil {
					for _, c := range vset.Doc.List {
						meta.Comments = append(meta.Comments, c.Text)
					}
				}

				// inline comment on the same line (e.g. var x = 5 // some note)
				if vset.Comment != nil {
					for _, c := range vset.Comment.List {
						meta.Comments = append(meta.Comments, c.Text)
					}
				}

				variables = append(variables, meta)
			}
		}
		return true
	})

	return variables, nil
}
