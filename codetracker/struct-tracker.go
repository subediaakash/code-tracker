package codetracker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type FieldMeta struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Tag      string   `json:"tag,omitempty"`
	Comments []string `json:"comments,omitempty"`
}

type StructMeta struct {
	Name       string      `json:"name"`
	Fields     []FieldMeta `json:"fields,omitempty"`
	Methods    []string    `json:"methods,omitempty"`
	Comments   []string    `json:"comments,omitempty"`
	IsExported bool        `json:"is_exported"`
}

func StructTracker(filePath string) ([]StructMeta, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	structs := make(map[string]*StructMeta)

	// pass 1: collect struct declarations
	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			meta := &StructMeta{
				Name:       typeSpec.Name.Name,
				IsExported: ast.IsExported(typeSpec.Name.Name),
			}

			// doc comment above the type declaration
			if genDecl.Doc != nil {
				for _, c := range genDecl.Doc.List {
					meta.Comments = append(meta.Comments, c.Text)
				}
			}
			if typeSpec.Comment != nil {
				for _, c := range typeSpec.Comment.List {
					meta.Comments = append(meta.Comments, c.Text)
				}
			}

			// fields
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					typeStr := fmt.Sprintf("%v", field.Type)
					tag := ""
					if field.Tag != nil {
						tag = strings.Trim(field.Tag.Value, "`")
					}

					var fieldComments []string
					if field.Doc != nil {
						for _, c := range field.Doc.List {
							fieldComments = append(fieldComments, c.Text)
						}
					}
					if field.Comment != nil {
						for _, c := range field.Comment.List {
							fieldComments = append(fieldComments, c.Text)
						}
					}

					if len(field.Names) == 0 {
						// embedded struct
						meta.Fields = append(meta.Fields, FieldMeta{
							Name:     typeStr,
							Type:     typeStr,
							Tag:      tag,
							Comments: fieldComments,
						})
					} else {
						for _, name := range field.Names {
							meta.Fields = append(meta.Fields, FieldMeta{
								Name:     name.Name,
								Type:     typeStr,
								Tag:      tag,
								Comments: fieldComments,
							})
						}
					}
				}
			}

			structs[meta.Name] = meta
		}
		return true
	})

	// pass 2: find methods (functions with a receiver matching a known struct)
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			return true
		}

		receiverType := extractReceiverType(fn.Recv)
		if s, found := structs[receiverType]; found {
			s.Methods = append(s.Methods, fn.Name.Name)
		}

		return true
	})

	result := make([]StructMeta, 0, len(structs))
	for _, s := range structs {
		result = append(result, *s)
	}
	return result, nil
}

// extractReceiverType returns the base type name from a receiver field list.
// Handles both value receivers (s MyStruct) and pointer receivers (s *MyStruct).
func extractReceiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}
