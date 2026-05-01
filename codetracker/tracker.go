package codetracker

import (
	"fmt"
	"strings"
)

func CodeTracker(filePath string) {
	imports, err := ImportTracker(filePath)
	if err != nil {
		fmt.Println("Error tracking imports:", err)
		return
	}

	variables, err := TrackVariables(filePath)
	if err != nil {
		fmt.Println("Error tracking variables:", err)
		return
	}

	functions, err := FunctionTracker(filePath, variables)
	if err != nil {
		fmt.Println("Error tracking functions:", err)
		return
	}

	structs, err := StructTracker(filePath)
	if err != nil {
		fmt.Println("Error tracking structs:", err)
		return
	}

	callGraph, err := CallGraphTracker(filePath)
	if err != nil {
		fmt.Println("Error tracking call graph:", err)
		return
	}

	fmt.Println("=== Imports ===")
	for _, imp := range imports {
		kind := "stdlib"
		if !imp.IsStdLib {
			kind = "external"
		}
		usedLabel := "used"
		if !imp.IsUsed {
			usedLabel = "NOT USED"
		}
		fmt.Printf("  %-40s [%-8s]  %s\n", imp.Path, kind, usedLabel)
		for _, c := range imp.Comments {
			fmt.Println("   ", c)
		}
	}

	fmt.Println("\n=== Variables ===")
	for _, v := range variables {
		typeLabel := v.Type
		if typeLabel == "" {
			typeLabel = "inferred"
		}
		fmt.Printf("  %s (%s) = %v  [line %d]\n", v.Name, typeLabel, v.Value, v.Line)
		for _, c := range v.Comments {
			fmt.Println("   ", c)
		}
	}

	fmt.Println("\n=== Functions ===")
	for _, f := range functions {
		fmt.Printf("  %s  [lines %d-%d]\n", f.Name, f.CalledAtLineNumber, f.EndedAtLineNumber)

		if len(f.Comments) > 0 {
			fmt.Println("    Comments:")
			for _, c := range f.Comments {
				fmt.Println("     ", c)
			}
		}

		if len(f.Parameters) > 0 {
			fmt.Println("    Parameters:")
			for _, p := range f.Parameters {
				fmt.Printf("      %s %s\n", p.Name, p.ObjectType)
			}
		}

		fmt.Printf("    Returns: %s\n", f.ReturnType)

		if len(f.VariablesUsed) > 0 {
			fmt.Println("    Variables used:")
			for _, v := range f.VariablesUsed {
				fmt.Printf("      %s\n", v.Name)
			}
		}
	}

	fmt.Println("\n=== Structs ===")
	for _, s := range structs {
		exported := ""
		if s.IsExported {
			exported = " (exported)"
		}
		fmt.Printf("  %s%s\n", s.Name, exported)
		for _, c := range s.Comments {
			fmt.Println("   ", c)
		}
		if len(s.Fields) > 0 {
			fmt.Println("    Fields:")
			for _, f := range s.Fields {
				tag := ""
				if f.Tag != "" {
					tag = fmt.Sprintf("  `%s`", f.Tag)
				}
				fmt.Printf("      %s %s%s\n", f.Name, f.Type, tag)
				for _, c := range f.Comments {
					fmt.Println("       ", c)
				}
			}
		}
		if len(s.Methods) > 0 {
			fmt.Println("    Methods:", s.Methods)
		}
	}

	fmt.Println("\n=== Call Graph ===")
	for _, entry := range callGraph {
		if len(entry.Calls) == 0 {
			fmt.Printf("  %s → (no calls)\n", entry.FunctionName)
			continue
		}
		fmt.Printf("  %s →\n", entry.FunctionName)
		for _, call := range entry.Calls {
			if call.Package != "" {
				fmt.Printf("      %s.%s  [line %d]\n", call.Package, call.Name, call.Line)
			} else {
				fmt.Printf("      %s  [line %d]\n", call.Name, call.Line)
			}
		}
	}
}

func PrintProjectGraph(graph *ProjectGraph) {
	fmt.Printf("Project root: %s\n", graph.Root)
	fmt.Printf("Packages found: %d\n\n", len(graph.Packages))

	for pkg, files := range graph.Packages {
		fmt.Printf("package %s  (%d file(s))\n", pkg, len(files))
		fmt.Println(strings.Repeat("-", 50))

		for _, file := range files {
			fmt.Printf("\n  file: %s\n", file.Path)

			if len(file.Imports) > 0 {
				fmt.Println("  Imports:")
				for _, imp := range file.Imports {
					kind := "stdlib"
					if !imp.IsStdLib {
						kind = "external"
					}
					used := "used"
					if !imp.IsUsed {
						used = "NOT USED"
					}
					fmt.Printf("    %-40s [%-8s] %s\n", imp.Path, kind, used)
				}
			}

			if len(file.Variables) > 0 {
				fmt.Println("  Variables:")
				for _, v := range file.Variables {
					typeLabel := v.Type
					if typeLabel == "" {
						typeLabel = "inferred"
					}
					fmt.Printf("    %s (%s) = %v  [line %d]\n", v.Name, typeLabel, v.Value, v.Line)
				}
			}

			if len(file.Structs) > 0 {
				fmt.Println("  Structs:")
				for _, s := range file.Structs {
					exported := ""
					if s.IsExported {
						exported = " (exported)"
					}
					fmt.Printf("    %s%s — %d field(s), %d method(s)\n", s.Name, exported, len(s.Fields), len(s.Methods))
				}
			}

			if len(file.Functions) > 0 {
				fmt.Println("  Functions:")
				for _, f := range file.Functions {
					fmt.Printf("    %s  [lines %d-%d]  returns %s\n", f.Name, f.CalledAtLineNumber, f.EndedAtLineNumber, f.ReturnType)
				}
			}

			if len(file.CallGraph) > 0 {
				fmt.Println("  Call Graph:")
				for _, entry := range file.CallGraph {
					if len(entry.Calls) == 0 {
						fmt.Printf("    %s → (no calls)\n", entry.FunctionName)
						continue
					}
					fmt.Printf("    %s →\n", entry.FunctionName)
					for _, call := range entry.Calls {
						if call.Package != "" {
							fmt.Printf("        %s.%s  [line %d]\n", call.Package, call.Name, call.Line)
						} else {
							fmt.Printf("        %s  [line %d]\n", call.Name, call.Line)
						}
					}
				}
			}
		}
		fmt.Println()
	}

	if len(graph.CrossFileCalls) > 0 {
		fmt.Println("=== Cross-File Calls ===")
		for _, c := range graph.CrossFileCalls {
			fmt.Printf("  %s::%s → %s::%s  [line %d]\n",
				c.CallerFile, c.CallerFunction,
				c.CalleeFile, c.CalleeFunction,
				c.Line,
			)
		}
	}
}
