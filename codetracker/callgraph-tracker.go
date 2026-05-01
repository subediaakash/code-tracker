package codetracker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type CallInfo struct {
	Name    string `json:"name"`
	Package string `json:"package,omitempty"`
	Line    int    `json:"line"`
}

type CallGraphMeta struct {
	FunctionName string     `json:"function"`
	Calls        []CallInfo `json:"calls,omitempty"`
}

func CallGraphTracker(filePath string) ([]CallGraphMeta, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var graph []CallGraphMeta

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		entry := CallGraphMeta{
			FunctionName: fn.Name.Name,
		}

		if fn.Body != nil {
			ast.Inspect(fn.Body, func(bn ast.Node) bool {
				callExpr, ok := bn.(*ast.CallExpr)
				if !ok {
					return true
				}

				line := fset.Position(callExpr.Pos()).Line
				switch fun := callExpr.Fun.(type) {
				case *ast.Ident:
					// local call: foo()
					entry.Calls = append(entry.Calls, CallInfo{Name: fun.Name, Line: line})

				case *ast.SelectorExpr:
					// pkg.Func() or obj.Method()
					pkg := fmt.Sprintf("%v", fun.X)
					entry.Calls = append(entry.Calls, CallInfo{
						Name:    fun.Sel.Name,
						Package: pkg,
						Line:    line,
					})
				}

				return true
			})
		}

		graph = append(graph, entry)
		return true
	})

	return graph, nil
}
