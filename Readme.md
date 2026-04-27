# leanBuildDocker — MVP

> Concise specification of the first deliverable version.

This document consolidates **what we're building, why, and how**. It is the single source of truth — no companion spec files. Read this first to get oriented.

---

## 1. The problem in two paragraphs

Most Docker images shipped today are 5 to 20 times larger than necessary. The reasons are repetitive and well-known: missing `.dockerignore`, oversized base images, no multi-stage build, dev dependencies in production, bad layer ordering, no cleanup in `RUN` instructions, running as root. Fixing these is mechanical work, but the knowledge is unevenly distributed — concentrated in senior DevOps engineers and absent from most application developers.

Existing tooling doesn't close this gap. Dockerfile linters require an existing Dockerfile. Buildpacks generate opaque images, not Dockerfiles. Image optimizers like `docker-slim` need a runnable application and dynamic analysis. LLMs hallucinate and aren't deterministic. Nobody is doing **static analysis of source code → optimized Dockerfile generation → human-readable explanation**, in an open, auditable, language-aware way.

## 2. The proposed solution

`leanBuildDocker` (CLI: `lbd`) is a deterministic, open-source command-line tool that:

1. **Reads** the project directory and its manifests (no execution).
2. **Infers** language, dependencies, system requirements, and execution details.
3. **Plans** an optimized build strategy: base images, multi-stage layout, install commands, user, ports.
4. **Generates** a Dockerfile, `.dockerignore`, and a `.lbd-report.md` explaining every decision.
5. **Optionally validates** by building the image and running `hadolint` on the output.

The user runs `lbd` in a project root and gets working, optimized artifacts in seconds — plus a written explanation of why each choice was made.

## 3. MVP scope: Go only

The first release supports **Go projects only**. Three reasons drive this:

- Go is where containerization gains are most dramatic (800 MB → ~18 MB with `scratch`).
- Go's ecosystem is exceptionally well-suited to static analysis: `go.mod` is declarative, no `setup.py`-style executable manifests, no `postinstall` scripts.
- Building the tool itself in Go alongside Go-project analysis creates a virtuous learning loop.

Node.js and Python are deferred to v0.4+ once the core pipeline is proven. The architecture is **prepared** for them — not exercised by them.

## 4. Guiding principles

These are decisions made once so they don't have to be re-litigated per feature:

- **Static analysis only.** No code execution. Safe in CI, deterministic, fast.
- **Explicit over implicit.** We produce a Dockerfile (readable, editable, version-controllable), not a built image.
- **Pedagogical by design.** Every decision is explained in `.lbd-report.md`. Users get smarter at Docker over time.
- **Safe defaults, opt-in aggressive minimization.** First-run output should "just work." Users opt into smaller-but-riskier outputs explicitly via `--minimize`.
- **Fail loudly with clear messages.** When the tool can't handle a project (e.g., complex monorepo without `--target`), it says so concretely and tells the user how to help it. No silent guessing.
- **Extensibility prepared, not premature.** The plugin interface for languages exists from day one, but only one implementation (Go) ships in MVP.

## 5. Architecture (conceptual)

The tool is a **pipeline** with discrete stages, each producing a structured artifact for the next:

```
Project ──► Scanner ──► Detector ──► Analyzer ──► Planner ──► Renderer ──► Files
                                          │
                                  ┌───────┴───────┐
                                  │   Go plugin   │  ◄─ only one in MVP
                                  └───────────────┘
                                          │
                                  ┌───────▼────────────┐
                                  │ System Package KB  │
                                  │   (embedded YAML)  │
                                  └────────────────────┘
```

### Stage responsibilities

- **Scanner** — Walks the project directory, builds a file inventory, applies early ignore rules.
- **Detector** — Identifies the project's language via marker files (`go.mod`). Returns a confidence score; in MVP, anything but Go yields a clear "not supported yet" exit.
- **Analyzer** — Language-specific plugin. Reads `go.mod`, `go.sum`, source files. Detects CGO usage, build tags, vendoring, multi-`cmd` layouts. Produces a `ProjectInfo`.
- **Planner** — Language-agnostic. Takes `ProjectInfo`, produces a `BuildPlan`: chosen base images, install steps, user, ports, multi-stage layout.
- **Renderer** — Takes `BuildPlan`, fills text templates, writes output files.
- **Validator** *(optional, `--validate`)* — Builds the generated Dockerfile, runs `hadolint`, optionally smoke-tests by running the container briefly.

### The two core data contracts

These structures are the system's spine. Everything else is plumbing.

- **`ProjectInfo`** — Universal description of a project. Output of any Analyzer. Contains: language, version, dependencies, system requirements, execution hints, exposed ports, environment variables, and a typed language-specific extension. Represents **facts**.

- **`BuildPlan`** — All decisions needed to render a Dockerfile. Output of the Planner. Contains: base image choices (build + runtime), system package install commands, user, layer order, optimization flags. Represents **decisions**.

### Why an interface for plugins, even with one implementation

The `Analyzer` interface is ~40 lines of code. It forces the Planner and Renderer to never depend on Go-specific types. When Node and Python are added later, no core code changes — only new plugins. This is a small upfront cost (≈2% of MVP code) for a guarantee of clean separation.

### Knowledge Base

System-level package mappings (e.g., "this dependency requires `libpq` at runtime") live in an embedded YAML file. Curated entries grow via PRs. In MVP, the KB starts small with the most common Go cases (CGO + sqlite, CGO + libc6, etc.).

## 6. What the user experiences

```bash
$ cd my-go-api
$ lbd
Detected: Go 1.23 (confidence: 1.0)
Analyzed: 14 direct dependencies, no CGO, no vendoring
Planned: multi-stage build, distroless/static runtime (estimated ~18 MB)
Wrote: Dockerfile, .dockerignore, .lbd-report.md

$ lbd --validate
[... same as above ...]
Building image... ok
Running smoke test... ok
Final size: 17.4 MB
Linting with hadolint... ok (0 warnings)
```

When something can't be inferred:

```bash
$ lbd
Detected: Go monorepo (3 cmd/* binaries found)
Cannot infer which service to containerize.
Run: lbd --target ./cmd/api  (or another path)
```

## 7. What's deliberately out of scope for the MVP

- Languages other than Go.
- Monorepo auto-resolution (user must point `--target` at a specific service).
- Dynamic analysis (running the application to observe behavior).
- Modifying existing Dockerfiles (we generate, never edit).
- Kubernetes manifests, Helm charts, image signing, SBOM generation.
- IDE/editor integrations (CLI-first; integrations come later).
- Telemetry, cloud features, configuration servers.

These are documented exclusions, not oversights. Each one we add later requires explicit justification.

## 8. Definition of "done" for the MVP

The MVP ships when:

1. `lbd` runs against a typical Go project (Gin/Echo/Chi API with a database) and produces a Dockerfile that builds cleanly and runs correctly.
2. The generated image is in the **15–30 MB range** for a standard Go API without CGO.
3. `lbd --validate` builds, smoke-tests, and lints the output successfully on a CI runner.
4. `.lbd-report.md` explains every decision in plain language.
5. A Go developer with no Docker expertise can use the tool successfully without reading source code.
6. `lbd doctor` provides actionable diagnostics for the common failure modes (CGO without correct base, missing certificates, runtime user issues).
7. The repository contains: the binary, templates, embedded KB, README, this document, and `CONTRIBUTING.md`.

## 9. Known risks and how we address them

| Risk                                         | Mitigation                                                                         |
| -------------------------------------------- | ---------------------------------------------------------------------------------- |
| User generates image that crashes at runtime | `--validate` includes smoke test; `lbd doctor` for diagnostics; safe defaults     |
| Monorepo confusion                           | Detect and fail loudly with `--target` instruction                                 |
| Outdated knowledge base                      | KB versioned alongside code; periodic CI checks against current Go releases        |
| Adoption friction vs. LLMs                   | Position as CI-first; emphasize determinism and auditability over conversation     |
| Maintenance burden                           | Narrow scope (Go only); no SLA promised; templates as data, not code               |

---

## 10. Next steps

In order:

1. **Define core interfaces in code** — `Analyzer`, `ProjectInfo`, `BuildPlan`. These structs and interfaces *are* the spec; writing them in Go forces precision that prose can't.
2. **Implement the Go Analyzer** — read `go.mod`/`go.sum`, detect CGO, vendoring, `cmd/` layout. Produce a real `ProjectInfo` from a real project.
3. **Implement Planner + Renderer skeleton** — consume `ProjectInfo`, emit `Dockerfile`, `.dockerignore`, `.lbd-report.md`.
4. **Wire the CLI** — `lbd` command running Scanner → Detector → Analyzer → Planner → Renderer end-to-end.
5. **Validate against 3–5 real Go projects** — fix gaps discovered in practice, not in speculation.
6. **Write ADRs for the 2–3 highest-stakes decisions** (static-only stance, plugin interface shape, base image selection logic) — after the code, when the real trade-offs are known.

This document is the single source of truth for "what we're building." If a future change conflicts with anything written here, this document is updated first, deliberately, before any code changes.