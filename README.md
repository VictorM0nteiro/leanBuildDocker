# leanBuildDocker

> A deterministic CLI that statically analyzes a Go project and generates an optimized `Dockerfile`, `.dockerignore`, and decision report — no code execution.

```bash
$ lbd
Detected: Go 1.23 (confidence: 1.0)
Analyzed: 2 direct dependencies, no CGO, no vendoring
Wrote: Dockerfile, .dockerignore, .lbd-report.md
```

The resulting image for a typical Go API lands in the **15–30 MB** range using `distroless/static`, instead of the 800 MB+ common in naive Dockerfiles.

## Install

```bash
go install github.com/VictorM0nteiro/leanBuildDocker/cmd/lbd@latest
```

Or grab a prebuilt binary from [Releases](https://github.com/VictorM0nteiro/leanBuildDocker/releases).

## Usage

```bash
# Generate a Dockerfile in the current project
lbd

# Point at a different directory
lbd --target ./services/api

# Overwrite existing artifacts
lbd --force

# Build the image and run a smoke test (requires Docker)
lbd --validate

# Diagnose an existing project + Dockerfile
lbd doctor
```

Main flags:

| Flag | Description |
|---|---|
| `--target`, `-t` | Project directory (default: cwd) |
| `--force`, `-f` | Overwrite existing `Dockerfile` / `.dockerignore` / `.lbd-report.md` |
| `--validate` | After generating, run `docker build` + smoke test |
| `--keep` | Keep the validation image instead of removing it |
| `--verbose`, `-v` | Verbose logging |

## How it works

A 5-stage pipeline; each stage only reads the previous stage's typed output:

```
Scanner → Detector → Analyzer → Planner → Renderer
                         ↑
                   Knowledge Base (embedded YAML)
```

- **No CGO + no vendoring:** `golang:<ver>-alpine` for build, `distroless/static` for runtime.
- **CGO required:** `golang:<ver>` (Debian, with gcc) for build, `distroless/base` (with glibc) for runtime.
- **Vendoring detected:** adds `-mod=vendor` and copies `vendor/`.
- **System packages:** dependencies declared in `internal/knowledge/packages.yaml` (e.g., `go-sqlite3` → `libsqlite3-dev`).

Every non-trivial decision is recorded in `.lbd-report.md` with rationale and alternatives considered.

## Scope

The MVP supports **Go only**. Node.js and Python are prepared for in the architecture via the `Analyzer` interface but are not implemented.

See [docs/MVP.md](docs/MVP.md) for the full problem statement, motivation, and definition of done.

## Development

```bash
make build       # binary at ./lbd
make test        # go test ./...
make lint        # go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

MIT — see [LICENSE](LICENSE).