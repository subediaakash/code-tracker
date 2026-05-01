package codetracker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type FunctionMetaData struct {
	Name               string            `json:"name"`
	Parameters         []ParameterObject `json:"parameters,omitempty"`
	ReturnType         string            `json:"return_type"`
	Comments           []string          `json:"comments,omitempty"`
	CalledAtLineNumber int               `json:"line_start"`
	EndedAtLineNumber  int               `json:"line_end"`
	VariablesUsed      []VariableMeta    `json:"variables_used,omitempty"`
}

type ParameterObject struct {
	Name       string `json:"name"`
	ObjectType string `json:"type"`
}

func FunctionTracker(filePath string, variablesUsed []VariableMeta) ([]FunctionMetaData, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []FunctionMetaData

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			meta := FunctionMetaData{
				Name:               fn.Name.Name,
				CalledAtLineNumber: fset.Position(fn.Pos()).Line,
				EndedAtLineNumber:  fset.Position(fn.End()).Line,
				VariablesUsed:      []VariableMeta{},
			}

			// Extract Parameters
			if fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {
					for _, name := range field.Names {
						meta.Parameters = append(meta.Parameters, ParameterObject{
							Name:       name.Name,
							ObjectType: fmt.Sprintf("%v", field.Type),
						})
					}
				}
			}

			// Extract Return Type
			if fn.Type.Results != nil {
				meta.ReturnType = fmt.Sprintf("%v", fn.Type.Results.List[0].Type)
			} else {
				meta.ReturnType = "void"
			}

			// Identify which variables from variablesUsed are referenced in this function body
			if fn.Body != nil {
				ast.Inspect(fn.Body, func(bn ast.Node) bool {
					if ident, ok := bn.(*ast.Ident); ok {
						for _, v := range variablesUsed {
							if ident.Name == v.Name {
								// Avoid duplicates
								exists := false
								for _, used := range meta.VariablesUsed {
									if used.Name == v.Name {
										exists = true
										break
									}
								}
								if !exists {
									meta.VariablesUsed = append(meta.VariablesUsed, v)
								}
							}
						}
					}
					return true
				})
			}

			// Extract doc comments above the function
			if fn.Doc != nil {
				for _, comment := range fn.Doc.List {
					meta.Comments = append(meta.Comments, comment.Text)
				}
			}

			// Extract inline comments inside the function body
			if fn.Body != nil {
				bodyStart := fn.Body.Lbrace
				bodyEnd := fn.Body.Rbrace
				for _, cg := range node.Comments {
					if cg.Pos() > bodyStart && cg.End() < bodyEnd {
						for _, c := range cg.List {
							meta.Comments = append(meta.Comments, c.Text)
						}
					}
				}
			}

			functions = append(functions, meta)
		}
		return true
	})

	return functions, nil
}
