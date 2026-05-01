package main

import (
	"flag"
	"fmt"
	"os"

	codetracker "github.com/subediaakash/code-tracker/codetracker"
)

const usage = `code-tracker — Go codebase analyser for humans and LLMs

Usage:
  code-tracker [--path <dir>] [--format <fmt>]

Flags:
  --path    Path to the Go project root (default: current directory)
  --format  Output format: pretty | json | ai  (default: pretty)
            pretty  human-readable tree
            json    full graph as indented JSON
            ai      compact markdown summary optimised for LLM token usage

Examples:
  code-tracker --path ./myproject --format ai
  code-tracker --path . --format json > graph.json
  code-tracker --format pretty
`

func main() {
	path := flag.String("path", ".", "path to the Go project root")
	format := flag.String("format", "pretty", "output format: pretty | json | ai")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	graph, err := codetracker.TrackProject(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		out, err := graph.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "json error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)

	case "ai":
		fmt.Println(graph.ToAISummary())

	case "pretty":
		codetracker.PrintProjectGraph(graph)

	default:
		fmt.Fprintf(os.Stderr, "unknown format %q — choose: pretty | json | ai\n", *format)
		os.Exit(1)
	}
}
