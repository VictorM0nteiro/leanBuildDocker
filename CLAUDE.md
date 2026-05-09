# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

`leanBuildDocker` (CLI: `lbd`) is a deterministic CLI tool that statically analyzes a Go project and generates an optimized `Dockerfile`, `.dockerignore`, and `.lbd-report.md`. No code is executed — analysis is purely static. MVP scope is Go only.

## Commands

Once the Go module exists, the standard commands will be:

```bash
make build       # go build ./cmd/lbd
make test        # go test ./...
make lint        # go vet ./... (+ golangci-lint if configured)
make clean       # remove build artifacts

# Run a single test
go test ./internal/analyzer/golang/... -run TestName -v

# Run lbd against a project
go run ./cmd/lbd [--validate] [--target ./cmd/api] [--verbose]
```

## Architecture

The tool is a **pipeline** — each stage has a single responsibility and communicates only via typed structs. No stage imports another stage's package; they share only `internal/types`.

```
Scanner → Detector → Analyzer → Planner → Renderer → Files
                         ↑
                   Go plugin (Strategy pattern)
                         ↑
                   Knowledge Base (embedded YAML)
```

### The two core contracts (spine of the system)

- **`ProjectInfo`** (`internal/types/`) — facts about the project: language, version, dependencies, CGO, vendoring, entry points, warnings. Output of any Analyzer.
- **`BuildPlan`** (`internal/types/`) — decisions needed to render a Dockerfile: base images, build/run commands, system packages, user, ports, `Decision` entries for the report. Output of the Planner.

### Key design rules

- **`cmd/lbd/main.go` must be dumb.** It sequences stage calls and passes outputs. Any conditional logic or rule evaluation found there belongs in a pipeline stage.
- **Analyzers implement an interface.** `internal/analyzer/golang/` is the only implementation in MVP, but the `Analyzer` interface in `internal/analyzer/` must never leak Go-specific types into the Planner or Renderer.
- **`BuildPlan.Decisions []Decision`** is the direct source for `.lbd-report.md`. Every non-obvious Planner choice must append a `Decision{Topic, Chose, Because, Alternatives}`.
- **Templates are data, not code.** Dockerfile generation lives in `templates/golang/dockerfile.tmpl` using `text/template`. The Renderer writes all files atomically — nothing touches disk until all templates render successfully.
- **Renderer refuses to overwrite** existing files unless `--force` is passed.

### Planner decision rules (Go)

| Condition | Build base | Runtime base |
|---|---|---|
| No CGO, no vendoring | `golang:<ver>-alpine` | `distroless/static` |
| CGO required | `golang:<ver>` (Debian) | `distroless/base` |
| Vendoring | Add `-mod=vendor` to build cmd, COPY `vendor/` |

Build flags for no-CGO: `go build -ldflags="-w -s" -trimpath -o /out/<bin> <entry-point>`

### Knowledge Base

`internal/knowledge/packages.yaml` is embedded via `go:embed`. It maps Go dependency paths to required system packages (build-time and runtime). The Analyzer flags dependencies; the Planner consumes those flags to add install steps.

## Dependencies (intentionally minimal)

- `github.com/spf13/cobra` — CLI
- `golang.org/x/mod/modfile` — `go.mod` parsing
- `gopkg.in/yaml.v3` — KB parsing

Adding any dependency beyond this list requires a one-paragraph justification. The tool tells users to ship small images; its own binary must reflect that.

## Error conventions

- Wrap errors: `fmt.Errorf("context: %w", err)`
- Errors reaching the CLI surface as plain-language messages, never Go stack traces
- Structured logging via `log/slog`; verbosity via `--verbose`
- Monorepo without `--target`: fail loudly with the exact message: `"Detected: Go monorepo (N cmd/* binaries found)\nCannot infer which service to containerize.\nRun: lbd --target ./cmd/<name>"`

## Development phases (current state)

See [DEVELOPMENT_PLAN.md](docs/DEVELOPMENT_PLAN.md) for the full roadmap. Check [docs/MVP.md](docs/MVP.md) for the problem statement and MVP definition.

Phase order: 0 (foundations) → 1 (dummy pipeline) → 2 (real Go analyzer) → 3 (planner) → 4 (renderer/templates) → 5 (KB) → 6 (--validate) → 7 (lbd doctor) → 8 (release).
