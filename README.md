# code-tracker

A CLI tool that analyses Go codebases and produces structured reports of every code element — imports, variables, functions, structs, and cross-file call graphs. Designed equally for human exploration and as context preparation for LLMs.

## Install

**From source (requires Go 1.23.2+):**

```bash
go install github.com/subediaakash/code-tracker@latest
```

**Or build manually:**

```bash
git clone https://github.com/subediaakash/go-codebase-tracker.git
cd go-codebase-tracker
go build -o code-tracker .
```

## Usage

```
code-tracker [--path <dir>] [--format <fmt>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | `.` | Path to the Go project root |
| `--format` | `pretty` | Output format: `pretty`, `json`, or `ai` |

## Output Formats

### `pretty` — Human-readable tree

Prints a structured tree showing each file's imports, globals, functions, and structs with full signatures and comments.

```bash
code-tracker --path ./myproject
```

```
=== PROJECT GRAPH ===

--- File: main.go ---
  IMPORTS:
    fmt (stdlib) [used]
    os  (stdlib) [used]
  FUNCTIONS:
    main()
      Calls: flag.Parse, codetracker.TrackProject, graph.ToJSON, ...
...
```

### `json` — Full structured graph

Outputs the complete code graph as indented JSON. Useful for tooling, editors, or any programmatic consumer.

```bash
code-tracker --path . --format json > graph.json
```

```json
{
  "files": [
    {
      "path": "main.go",
      "imports": [...],
      "functions": [...],
      "structs": [...],
      "variables": [...]
    }
  ],
  "callGraph": { ... }
}
```

### `ai` — Compact markdown for LLMs

Produces a token-efficient markdown summary of the codebase. Use this when you want to paste your project structure into an LLM prompt without burning context on repetitive boilerplate.

```bash
code-tracker --path ./myproject --format ai
```

```markdown
## Package: main
**File:** main.go

### Functions
- `main()` — entry point; parses flags, calls TrackProject, formats output

### Structs
...

### Cross-file calls
- main.go → codetracker/tracker.go: TrackProject
...
```

## Examples

```bash
# Analyse current directory, human-readable
code-tracker

# Analyse a specific project
code-tracker --path ~/projects/myapp

# Export full graph for tooling
code-tracker --path ~/projects/myapp --format json > myapp-graph.json

# Generate LLM context for a project
code-tracker --path ~/projects/myapp --format ai | pbcopy
```

## What Gets Analysed

- **Imports** — stdlib vs external, used vs unused
- **Functions** — signatures, parameters, return types, doc comments, intra-function variable usage
- **Structs** — fields, struct tags, methods, export status
- **Variables** — package-level globals with types and values
- **Call graph** — which functions call which, both within a file and across files

Test files (`*_test.go`) are intentionally excluded from analysis.

## No Dependencies

`code-tracker` uses only the Go standard library (`go/ast`, `go/parser`, `go/token`). No third-party packages.

## License

MIT
