package codetracker

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type FileSummary struct {
	Path      string             `json:"path"`
	Package   string             `json:"package"`
	Imports   []ImportMeta       `json:"imports,omitempty"`
	Variables []VariableMeta     `json:"variables,omitempty"`
	Functions []FunctionMetaData `json:"functions,omitempty"`
	Structs   []StructMeta       `json:"structs,omitempty"`
	CallGraph []CallGraphMeta    `json:"call_graph,omitempty"`
}

type CrossFileCall struct {
	CallerFile     string `json:"caller_file"`
	CallerFunction string `json:"caller_function"`
	CalleeFile     string `json:"callee_file"`
	CalleeFunction string `json:"callee_function"`
	Line           int    `json:"line"`
}

type ProjectGraph struct {
	Root           string               `json:"root"`
	Packages       map[string][]FileSummary `json:"packages"`
	CrossFileCalls []CrossFileCall      `json:"cross_file_calls,omitempty"`
}

func TrackProject(rootDir string) (*ProjectGraph, error) {
	graph := &ProjectGraph{
		Root:     rootDir,
		Packages: make(map[string][]FileSummary),
	}

	err := walkDir(rootDir, graph)
	if err != nil {
		return nil, err
	}

	graph.CrossFileCalls = resolveCrossFileCalls(graph)

	return graph, nil
}

// walkDir recursively walks a directory and processes every .go file it finds.
func walkDir(dir string, graph *ProjectGraph) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			// skip vendor, hidden dirs (.git, etc.)
			if name == "vendor" || strings.HasPrefix(name, ".") {
				continue
			}
			if err := walkDir(filepath.Join(dir, name), graph); err != nil {
				return err
			}
			continue
		}

		// only process .go files, skip test files
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(dir, name)
		summary, err := buildFileSummary(filePath)
		if err != nil {
			// skip unparseable files rather than stopping the whole walk
			continue
		}

		graph.Packages[summary.Package] = append(graph.Packages[summary.Package], summary)
	}

	return nil
}

// buildFileSummary runs all trackers on a single file and returns the combined result.
func buildFileSummary(filePath string) (FileSummary, error) {
	// parse once just to extract the package name
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return FileSummary{}, err
	}

	summary := FileSummary{
		Path:    filePath,
		Package: node.Name.Name,
	}

	summary.Imports, _ = ImportTracker(filePath)
	summary.Variables, _ = TrackVariables(filePath)
	summary.Functions, _ = FunctionTracker(filePath, summary.Variables)
	summary.Structs, _ = StructTracker(filePath)
	summary.CallGraph, _ = CallGraphTracker(filePath)

	return summary, nil
}

// resolveCrossFileCalls finds calls where FunctionA in file1 calls FunctionB defined in file2
// within the same package.
func resolveCrossFileCalls(graph *ProjectGraph) []CrossFileCall {
	// build map: package → functionName → filePath
	funcToFile := make(map[string]map[string]string)
	for pkg, files := range graph.Packages {
		funcToFile[pkg] = make(map[string]string)
		for _, file := range files {
			for _, fn := range file.Functions {
				funcToFile[pkg][fn.Name] = file.Path
			}
		}
	}

	var crossFileCalls []CrossFileCall

	for pkg, files := range graph.Packages {
		for _, file := range files {
			for _, entry := range file.CallGraph {
				for _, call := range entry.Calls {
					// package-prefixed calls are external — skip
					if call.Package != "" {
						continue
					}
					calleePath, exists := funcToFile[pkg][call.Name]
					if !exists {
						continue
					}
					// only record when the callee lives in a different file
					if calleePath != file.Path {
						crossFileCalls = append(crossFileCalls, CrossFileCall{
							CallerFile:     file.Path,
							CallerFunction: entry.FunctionName,
							CalleeFile:     calleePath,
							CalleeFunction: call.Name,
							Line:           call.Line,
						})
					}
				}
			}
		}
	}

	return crossFileCalls
}
