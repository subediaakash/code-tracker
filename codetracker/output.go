package codetracker

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// ToJSON returns the full ProjectGraph as indented JSON.
// Use this for tool integrations, APIs, or programmatic consumers.
func (g *ProjectGraph) ToJSON() (string, error) {
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToAISummary returns a compact markdown-style summary designed to minimise
// token usage when feeding codebase context to an LLM.
// It omits raw values and redundant detail, favouring structure and relationships.
func (g *ProjectGraph) ToAISummary() string {
	var b strings.Builder

	pkgNames := make([]string, 0, len(g.Packages))
	for pkg := range g.Packages {
		pkgNames = append(pkgNames, pkg)
	}

	b.WriteString(fmt.Sprintf("PROJECT: %s\n", g.Root))
	b.WriteString(fmt.Sprintf("PACKAGES: %s\n", strings.Join(pkgNames, ", ")))

	// build a cross-file call lookup: callerFile+callerFunc → []calleeFile::calleeFunc
	type xref struct{ calleeFile, calleeFunc string }
	xrefs := make(map[string][]xref)
	for _, c := range g.CrossFileCalls {
		key := c.CallerFile + "::" + c.CallerFunction
		xrefs[key] = append(xrefs[key], xref{c.CalleeFile, c.CalleeFunction})
	}

	for _, pkg := range pkgNames {
		files := g.Packages[pkg]
		b.WriteString(fmt.Sprintf("\n─── PACKAGE %s (%d file(s)) ───\n", pkg, len(files)))

		for _, file := range files {
			b.WriteString(fmt.Sprintf("\nFILE %s\n", file.Path))

			// imports — only show unused ones as a warning, list used ones compactly
			if len(file.Imports) > 0 {
				var used, unused []string
				for _, imp := range file.Imports {
					kind := "std"
					if !imp.IsStdLib {
						kind = "ext"
					}
					entry := fmt.Sprintf("%s(%s)", imp.LocalName, kind)
					if imp.IsUsed {
						used = append(used, entry)
					} else {
						unused = append(unused, imp.Path)
					}
				}
				if len(used) > 0 {
					b.WriteString(fmt.Sprintf("  IMPORTS: %s\n", strings.Join(used, ", ")))
				}
				if len(unused) > 0 {
					b.WriteString(fmt.Sprintf("  UNUSED IMPORTS: %s\n", strings.Join(unused, ", ")))
				}
			}

			// variables — name and type only, skip internal/generated vars
			if len(file.Variables) > 0 {
				parts := make([]string, 0, len(file.Variables))
				for _, v := range file.Variables {
					t := v.Type
					if t == "" {
						t = "inferred"
					}
					parts = append(parts, fmt.Sprintf("%s %s", v.Name, t))
				}
				b.WriteString(fmt.Sprintf("  VARS: %s\n", strings.Join(parts, ", ")))
			}

			// structs — compact single-line field list
			for _, s := range file.Structs {
				vis := "-"
				if s.IsExported {
					vis = "+"
				}
				doc := firstComment(s.Comments)
				if doc != "" {
					b.WriteString(fmt.Sprintf("  STRUCT %s %s  // %s\n", vis, s.Name, doc))
				} else {
					b.WriteString(fmt.Sprintf("  STRUCT %s %s\n", vis, s.Name))
				}

				fields := make([]string, 0, len(s.Fields))
				for _, f := range s.Fields {
					fields = append(fields, fmt.Sprintf("%s %s", f.Name, f.Type))
				}
				b.WriteString(fmt.Sprintf("    FIELDS: %s\n", strings.Join(fields, " | ")))

				if len(s.Methods) > 0 {
					b.WriteString(fmt.Sprintf("    METHODS: %s\n", strings.Join(s.Methods, ", ")))
				}
			}

			// functions
			for _, fn := range file.Functions {
				vis := "-"
				if isExportedName(fn.Name) {
					vis = "+"
				}

				params := formatParams(fn.Parameters)
				doc := firstComment(fn.Comments)

				b.WriteString(fmt.Sprintf("  FUNC %s %s(%s) → %s\n", vis, fn.Name, params, fn.ReturnType))
				if doc != "" {
					b.WriteString(fmt.Sprintf("    doc: %s\n", doc))
				}

				// inline calls — mark cross-file ones with their source file
				callKey := file.Path + "::" + fn.Name
				crossFileTargets := make(map[string]string) // funcName → file
				for _, x := range xrefs[callKey] {
					crossFileTargets[x.calleeFunc] = filepath.Base(x.calleeFile)
				}

				var callLines []string
				seen := make(map[string]bool)
				for _, cg := range file.CallGraph {
					if cg.FunctionName != fn.Name {
						continue
					}
					for _, call := range cg.Calls {
						label := call.Name
						if call.Package != "" {
							label = call.Package + "." + call.Name
						} else if f, ok := crossFileTargets[call.Name]; ok {
							label = call.Name + "[" + f + "]"
						}
						if !seen[label] {
							seen[label] = true
							callLines = append(callLines, label)
						}
					}
				}
				if len(callLines) > 0 {
					b.WriteString(fmt.Sprintf("    calls: %s\n", strings.Join(callLines, ", ")))
				}
			}
		}
	}

	if len(g.CrossFileCalls) > 0 {
		b.WriteString("\nCROSS-FILE CALLS:\n")
		for _, c := range g.CrossFileCalls {
			b.WriteString(fmt.Sprintf("  %s::%s → %s::%s\n",
				filepath.Base(c.CallerFile), c.CallerFunction,
				filepath.Base(c.CalleeFile), c.CalleeFunction,
			))
		}
	}

	return b.String()
}

// firstComment strips the leading // and returns the first non-empty comment.
func firstComment(comments []string) string {
	for _, c := range comments {
		text := strings.TrimSpace(strings.TrimPrefix(c, "//"))
		if text != "" {
			return text
		}
	}
	return ""
}

// formatParams turns []ParameterObject into "name type, name type".
func formatParams(params []ParameterObject) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s %s", p.Name, p.ObjectType))
	}
	return strings.Join(parts, ", ")
}

// isExportedName returns true if the name starts with an uppercase letter.
func isExportedName(name string) bool {
	if name == "" {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}
