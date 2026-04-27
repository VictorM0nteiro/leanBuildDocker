# Development Plan

> Phased roadmap to take `leanBuildDocker` from empty repository to working MVP.

This document is the operational counterpart to [`MVP.md`](./MVP.md). Where `MVP.md` defines **what** we're building, this document defines **how and in what order** we build it.

The plan is organized into **phases**. Each phase has a clear objective, concrete deliverables, a "done when" criterion, and notes on dependencies. Phases are sequential — Phase N+1 depends on Phase N being complete enough to support it.

Time estimates are **deliberately omitted**. This is a side project; pace will vary with availability. Scope per phase is fixed; pace is not.

---

### Architecture: The Pipeline Pattern

The core architectural style of `leanBuildDocker` is a **Pipeline**. Each stage does exactly one thing and passes its resulting data structure to the next.

#### Why a Pipeline and not other architectures?
Layered architectures, Clean Architecture, or Hexagonal Architecture would be massive over-engineering for this tool. Those patterns exist to manage mutable states, database integrations, complex business rules, and multiple ingress interfaces. `lbd` has none of these. 

A Pipeline is the natural, proven pattern for tools that transform a defined input into a defined output. Compilers, transpilers, and linters use this model. Your project is fundamentally a code-to-infrastructure transformation tool.

#### The 4 Design Patterns in Use
While the macro-architecture is a Pipeline, the internal implementation relies heavily on four specific patterns:

* **Strategy:** Each supported language is an analysis strategy. `GoAnalyzer` (and in the future, `NodeAnalyzer` or `PythonAnalyzer`) implements the exact same `Analyzer` interface. The core pipeline does not know or care which language is being processed.
* **Template Method:** The high-level skeleton of the generated Dockerfile is fixed, but the specific details (dependencies, build commands) vary. This maps perfectly to Go's standard `text/template` package.
* **Builder:** The `BuildPlan` data structure is complex and constructed in discrete steps (choosing a base image, adding system packages, configuring the user step-by-step).
* **Functional Options:** Used for configuring Go structs elegantly (e.g., `analyzer.New(WithStrictMode(true))`), keeping constructors clean and easily extensible in the future.

#### Repository Structure Map
`cmd/lbd/`         → Orchestrator (minimal lines of code, no business logic)
`internal/`
├── `types/`       → Core data contracts: ProjectInfo, BuildPlan
├── `scanner/`     → Builds file inventory and applies ignores
├── `detector/`    → Identifies language via marker files
├── `analyzer/`    → Interface + Language-specific plugins (e.g., golang)
├── `planner/`     → Decision engine (translates facts into Docker steps)
├── `renderer/`    → File writing and template execution
└── `knowledge/`   → System package mappings (YAML)
`templates/`       → dockerfile.tmpl, report.tmpl, etc.
`testdata/`        → Test projects (fixtures)

#### The Golden Rule
**Each pipeline stage has a single responsibility and complete isolation.** The stages do not know about each other; they only know about the `types` they consume and return. 

The orchestrator (`main.go` / CLI entrypoint) is intentionally "dumb" — it merely catches the output of one stage, checks for errors, and feeds the data to the next. **If you find yourself writing conditional business logic or rule evaluation in `cmd/lbd/`, the code is in the wrong place.**

---

## Phase 0 — Repository foundations

**Objective:** establish the home of the project so that all subsequent work has a place to land.

**Deliverables:**

- Initialize the Go module: `go mod init github.com/VictorM0nteiro/leanBuildDocker`
- Directory skeleton (matching the architecture in `MVP.md`):
  ```
  cmd/lbd/             # CLI entrypoint
  internal/
    scanner/
    detector/
    analyzer/
      golang/
    planner/
    renderer/
    knowledge/
  templates/
  testdata/
  docs/                # already populated
  ```
- `LICENSE` (MIT) at the root.
- Initial `.gitignore` for Go projects (binaries, coverage files, IDE folders).
- `Makefile` with placeholder targets: `build`, `test`, `lint`, `clean`.
- GitHub Actions workflow stub: runs `go build`, `go test`, `go vet` on every PR.
- First commit pushed to GitHub, repository public.

**Done when:** the repository is publicly visible, the CI runs and passes (with effectively no code yet), and the directory structure exists.

**Notes:** intentionally minimal. Resist the urge to set up complex tooling here — that comes when we have code that needs it.

---

## Phase 1 — CLI skeleton with dummy pipeline

**Objective:** prove the data flow from CLI input to file output works end-to-end before writing any real logic. This validates the architecture in code, not just in documents.

**Deliverables:**

- CLI wired with `spf13/cobra`. Root command `lbd`. Subcommands defined as stubs: `lbd`, `lbd doctor`, `lbd analyze`, `lbd explain`. Only `lbd` is functional in this phase; others print "not implemented yet."
- All five pipeline stages exist as Go packages with **dummy implementations** that return hardcoded data:
  - `scanner.Scan(path) (*FileInventory, error)` — returns a mock inventory.
  - `detector.Detect(*FileInventory) (Language, float64)` — always returns `(LanguageGo, 1.0)`.
  - `analyzer.Analyze(*FileInventory) (*ProjectInfo, error)` — returns a hardcoded `ProjectInfo` representing a fictional Go project.
  - `planner.Plan(*ProjectInfo) (*BuildPlan, error)` — returns a hardcoded `BuildPlan`.
  - `renderer.Render(*BuildPlan, outputDir) error` — writes a hardcoded "Hello World" Dockerfile.
- The orchestrator in `cmd/lbd/main.go` calls each stage in sequence, passing outputs between them.
- Logging via `log/slog` (Go 1.21+ standard) with `--verbose` flag.

**Done when:** running `lbd` in any directory produces a `Dockerfile` and a `.lbd-report.md` file. The content is fake, but the pipeline executed end-to-end and the user got files.

**Why this matters:** this phase catches integration issues (package layout, dependency direction, signature mismatches) when they're cheap to fix. By the end, the architecture exists in code — only the intelligence is missing.

### Skeleton sketch — `ProjectInfo` (Phase 1 version)

This is the minimal shape needed for the dummy pipeline. It will grow in Phase 2.

```go
// internal/analyzer/project_info.go (Phase 1 — minimal)
type ProjectInfo struct {
    Language        string   // "go" for now
    LanguageVersion string   // e.g. "1.23"
    Dependencies    []string // names only, no metadata yet
    EntryPoint      string   // e.g. "./cmd/api"
    HasCGO          bool
}
```

Just enough fields to write a Dockerfile that compiles. We don't try to be complete — we try to be **functional**.

### Skeleton sketch — `BuildPlan` (Phase 1 version)

```go
// internal/planner/build_plan.go (Phase 1 — minimal)
type BuildPlan struct {
    BuildBaseImage   string   // e.g. "golang:1.23-alpine"
    RuntimeBaseImage string   // e.g. "gcr.io/distroless/static-debian12:nonroot"
    BuildCommand     []string // exec form
    RunCommand       []string
    ExposedPort      int
    UseMultiStage    bool
}
```

Same philosophy — just enough to render a basic Dockerfile.

---

## Phase 2 — Real `ProjectInfo` for Go

**Objective:** replace the dummy Analyzer with a real one that parses actual Go projects. This is where the tool stops being a tube and starts being intelligent.

**Deliverables:**

- `internal/analyzer/golang/` populated with real parsing logic.
- Use `golang.org/x/mod/modfile` to parse `go.mod`. Extract: module name, Go version, direct dependencies with versions.
- Use `golang.org/x/mod/modfile` to parse `go.sum` for resolved versions when available.
- Source-level analysis to detect CGO: walk `.go` files, look for `import "C"` directives. Use `go/parser` (stdlib) — never regex.
- Detect vendoring: presence of `vendor/` directory.
- Detect multi-`cmd` layout: enumerate subdirectories of `cmd/` containing `main.go`.
- Detect entry point heuristic: prefer `cmd/<service>/main.go`, fall back to `main.go` at root.
- Expand `ProjectInfo` to include the new fields (see sketch below).
- Test fixtures under `testdata/projects/`: at least three real-ish Go projects for the analyzer to parse:
  - `simple-api/` — minimal Gin/Chi API, no CGO
  - `cgo-sqlite/` — uses `mattn/go-sqlite3`, requires CGO
  - `monorepo/` — multiple `cmd/*` binaries

**Done when:**
- Running the Analyzer against each fixture produces a `ProjectInfo` matching expected values.
- Unit tests cover `go.mod` parsing, CGO detection, and entry point detection.
- The dummy fallback in `Analyze()` is fully removed.

### Skeleton sketch — `ProjectInfo` (Phase 2 version)

```go
type ProjectInfo struct {
    // Identification
    Language        string
    LanguageVersion string         // resolved from go.mod
    ModulePath      string         // e.g. "github.com/user/project"
    ProjectRoot     string         // absolute path

    // Dependencies (Go-specific shape; we'll generalize when Node/Python join)
    DirectDependencies []GoDep
    
    // Build characteristics
    HasCGO          bool
    BuildTags       []string       // collected from `//go:build` comments if relevant
    UsesVendoring   bool

    // Layout
    MainPackages    []string       // e.g. ["./cmd/api", "./cmd/worker"]
    EntryPoint      string         // chosen target (./cmd/api by default)

    // Metadata
    Warnings        []Warning
}

type GoDep struct {
    Path     string  // e.g. "github.com/gin-gonic/gin"
    Version  string  // raw from go.mod
    Resolved string  // from go.sum
    Indirect bool    // marked `// indirect` in go.mod
}
```

Note: this is **Go-shaped**, not language-agnostic. We accept that. When Node arrives, we extract a common core and push Go-specific fields into a sub-struct. Until then, this shape is honest about what we know.

---

## Phase 3 — Planner with real Go decisions

**Objective:** convert `ProjectInfo` into a `BuildPlan` using rules that encode Docker expertise for Go projects specifically.

**Deliverables:**

- `internal/planner/golang.go` (or similar) with the decision rules:
  - **No CGO + no vendoring** → build base `golang:<version>-alpine`, runtime base `distroless/static`.
  - **CGO required** → build base `golang:<version>` (full Debian, has gcc), runtime base `distroless/base` (has glibc) or `alpine` if user prefers smaller.
  - **Vendoring used** → add `-mod=vendor` to build command, copy `vendor/` directory.
  - **Multi-`cmd` without `--target`** → fail loudly with the message format already defined in `MVP.md`.
- Build command construction: `go build -ldflags="-w -s" -trimpath -o /out/<bin> <entry-point>` for the no-CGO case; adjusted forms for CGO.
- Run command: `[/<bin>]` for static binaries; `["/usr/local/bin/<bin>"]` with appropriate path for dynamic.
- User selection: `nonroot` UID/GID for distroless; explicit `adduser` for Alpine paths.
- Port detection: extract from `EXPOSE` hints in source if present, else default to 8080 with a warning that this was a guess.
- The `BuildPlan` struct grows to support all of the above.

**Done when:**
- Each Phase 2 fixture produces a `BuildPlan` that matches expected output.
- Unit tests for the Planner cover at least: no-CGO happy path, CGO path, vendoring path, monorepo-without-target failure path.

### Skeleton sketch — `BuildPlan` (Phase 3 version)

```go
type BuildPlan struct {
    // Stage 1: build
    BuildStage Stage
    
    // Stage 2: runtime
    RuntimeStage Stage
    
    // Cross-stage
    UseMultiStage bool
    BinaryName    string      // e.g. "api"
    BinaryPath    string      // e.g. "/out/api" in build, "/api" in runtime
    
    // Runtime configuration
    User          string      // "nonroot" | "65532:65532"
    WorkingDir    string      // typically "/app" or "/"
    ExposedPorts  []int
    
    // For the report
    Decisions     []Decision  // human-readable rationale for each non-trivial choice
}

type Stage struct {
    BaseImage     string      // e.g. "golang:1.23-alpine"
    SystemPackages []string   // packages to install via apt/apk
    Commands      []Command   // ordered list of build steps
    CopySteps     []CopyStep  // what to COPY and from where
}

type Command struct {
    Args    []string  // exec form
    Comment string    // emitted as a Dockerfile comment for clarity
}

type Decision struct {
    Topic       string  // "runtime base image", "build flags"
    Chose       string  // "distroless/static"
    Because     string  // "no CGO detected, static binary"
    Alternatives []string // shown in the report for transparency
}
```

The `Decisions` slice is what feeds the `.lbd-report.md` directly. Every non-trivial choice the Planner made gets a `Decision` entry.

---

## Phase 4 — Renderer and templates

**Objective:** turn a `BuildPlan` into actual files on disk, using readable templates.

**Deliverables:**

- `templates/golang/dockerfile.tmpl` — Go template that consumes a `BuildPlan` and emits a Dockerfile.
- `templates/dockerignore.tmpl` — generic template for `.dockerignore`, parameterized by language.
- `templates/report.tmpl` — template for `.lbd-report.md`, iterates over `Decisions`.
- `internal/renderer/` package using Go's standard `text/template`.
- Templates support multi-stage Dockerfiles with proper indentation, comments, and `EXPOSE`/`USER`/`ENTRYPOINT` directives.
- The Renderer writes to disk only after all templates render successfully (no half-written outputs on error).
- The Renderer refuses to overwrite existing files unless `--force` is passed; this prevents accidental loss of hand-edited Dockerfiles.

**Done when:**
- Each Phase 2 fixture, run end-to-end, produces a Dockerfile that we manually verify (a) compiles via `docker build` and (b) yields an image in the expected size range.
- The `.lbd-report.md` is human-readable and explains every Planner decision.

---

## Phase 5 — System Package Knowledge Base (minimal)

**Objective:** lay the foundation for system-package mappings, even though Go's needs are modest compared to Node/Python.

**Deliverables:**

- `internal/knowledge/packages.yaml` embedded via `go:embed`.
- Initial entries for Go:
  - `github.com/mattn/go-sqlite3` → requires CGO + `sqlite-dev` (Alpine) / `libsqlite3-dev` (Debian) at build, no runtime system packages needed.
  - A handful of other CGO-dependent libraries (depends on what's common; can be empty initially).
- `ca-certificates` is always added when the project makes outbound HTTPS calls (default assumption: yes, unless we can prove otherwise).
- Loader code that parses the YAML at startup and exposes a lookup function.
- Documentation: `docs/KNOWLEDGE_BASE.md` (out of scope for this phase; deferred to its own document later) explains how to contribute entries.

**Done when:**
- The Analyzer can flag dependencies that require system packages.
- The Planner consumes those flags and adds the right install steps to the build stage.
- At least the SQLite case works end-to-end.

**Notes:** the KB will grow throughout the project's life. This phase establishes the mechanism, not the content.

---

## Phase 6 — `--validate` mode

**Objective:** add the safety net that gives users (and us) confidence that the generated Dockerfile actually works.

**Deliverables:**

- `lbd --validate` flag on the root command.
- After generation, the Validator:
  1. Runs `docker build -t lbd-validate-<hash> .` in the project directory.
  2. Captures and reports the final image size from `docker image inspect`.
  3. Runs `docker run --rm -d <image>`, waits 5 seconds, captures stderr, then stops the container. This is the **smoke test**.
  4. Runs `hadolint` against the generated Dockerfile (assumed installed; otherwise warns).
  5. Reports the full result in a structured summary.
- Pattern matching against known runtime errors in stderr:
  - `no such file or directory` after a binary path → likely missing `/etc/ssl/certs` or similar; suggest fix.
  - `cannot open shared object file` → likely libc mismatch (used Alpine but glibc needed); suggest using distroless base.
  - Exit code 0 immediately → app may have nothing to do; warn user.
- Cleanup: tagged validation images are removed after the run unless `--keep` is passed.

**Done when:**
- All Phase 2 fixtures pass `lbd --validate` cleanly.
- A deliberately broken fixture (e.g., CGO with wrong base) produces a clear, actionable error message via the smoke test.

---

## Phase 7 — `lbd doctor`

**Objective:** the diagnostic command users reach for when something goes wrong post-deployment.

**Deliverables:**

- `lbd doctor` subcommand. Two modes:
  - **Without arguments**: analyzes the current directory's project + existing Dockerfile (if present) and lists potential issues.
  - **With `--image <name>`**: inspects a built image and provides analysis (image size breakdown, base image used, user, port exposure, presence of common runtime issues).
- Checks performed:
  - Does the project use CGO but the Dockerfile uses `scratch`?
  - Are there `*_test.go` files in the build context (suggests `.dockerignore` problem)?
  - Is the runtime user root? (warn)
  - Is `EXPOSE` declared but the app appears to listen on a different port?
  - Is `ca-certificates` likely missing for an app that imports `net/http`?
- Output format: structured list of findings with severity (info/warn/error) and suggested fixes.

**Done when:**
- `lbd doctor` runs on each fixture and produces sensible findings (zero false positives on the clean fixtures).
- A documentation page (`docs/DOCTOR.md`, future) lists all checks the doctor performs.

---

## Phase 8 — Polish, packaging, first release

**Objective:** make the tool installable, documented, and announceable.

**Deliverables:**

- `goreleaser` configuration to build cross-platform binaries (Linux/macOS/Windows, amd64/arm64) on tagged commits.
- GitHub Releases automation: `git tag v0.1.0 && git push --tags` produces release artifacts.
- `go install` path verified: `go install github.com/VictorM0nteiro/leanBuildDocker/cmd/lbd@latest` works.
- README updated to reflect what actually shipped (vs. what was aspirational).
- `CONTRIBUTING.md` with: how to set up locally, how to add a fixture, how to contribute to the KB, code style notes.
- `CODE_OF_CONDUCT.md` (Contributor Covenant standard).
- A short blog post or LinkedIn post announcing the project, with a concrete before/after example.

**Done when:**
- The release exists publicly on GitHub.
- A user can install and use the tool without consulting the source code.
- Someone other than you has installed it and provided feedback (even if just a friend).

---

## Cross-phase concerns

Some concerns apply across all phases and are not "done" by any single phase. Tracking them here prevents them from being forgotten.

### Testing

- Every phase produces tests for the code it adds.
- Test types:
  - **Unit tests**: per package, fast, no I/O.
  - **Integration tests**: pipeline end-to-end against fixtures in `testdata/`.
  - **Validation tests** (Phase 6+): actual `docker build` runs in CI, gated to a separate workflow because they're slow.
- Coverage target: 70%+ on `internal/analyzer/` and `internal/planner/`. Lower on glue code is acceptable.

### Logging and errors

- All errors wrap with context using `fmt.Errorf("...: %w", err)`.
- Errors that reach the user (CLI exit) are printed in plain language, not Go stack traces.
- Internal logs use `log/slog` with structured fields. Verbosity controlled by `--verbose`.

### Documentation

- Documents in `docs/` evolve alongside code. When a phase introduces a concept that needs explanation (e.g., the KB structure in Phase 5), a corresponding `docs/*.md` is updated or created.
- Code comments follow Go conventions: package doc comment for every `internal/` package, exported symbols documented.

### Dependencies of `lbd` itself

We are deliberately conservative about adding dependencies to `lbd`'s own `go.mod`. Current planned set:
- `github.com/spf13/cobra` — CLI structure.
- `golang.org/x/mod/modfile` — parsing `go.mod`.
- `gopkg.in/yaml.v3` — knowledge base parsing.
- (Possibly) `github.com/charmbracelet/lipgloss` for prettier output — only if it earns its place.

Anything beyond this list requires a one-paragraph justification before adoption. We do not want a tool that tells people to ship small Docker images while shipping a bloated CLI itself.

---

## Phase dependency graph

```
Phase 0 ─► Phase 1 ─► Phase 2 ─► Phase 3 ─► Phase 4 ─► Phase 6 ─► Phase 7 ─► Phase 8
                                     │
                                     └─► Phase 5 ─┐
                                                  ▼
                                           (feeds back into 3 & 4)
```

- Phases 0 → 4 are strictly sequential. Each is a prerequisite for the next.
- Phase 5 (KB) can begin in parallel with Phase 4 once Phase 3 is done; its outputs feed into Phases 3 and 4 (Planner consumes KB; Renderer reflects KB-driven install steps).
- Phase 6 (validate) requires Phases 1–4 done.
- Phase 7 (doctor) is largely independent of Phase 6 but shares some inspection code.
- Phase 8 (release) requires everything else.

---

## What "MVP shipped" means

Re-stating from `MVP.md` in operational terms: the MVP is shipped when Phase 8 is complete. That means:

- A user runs `go install github.com/VictorM0nteiro/leanBuildDocker/cmd/lbd@latest`.
- They `cd` into a Go project (any of the fixture-style projects).
- They run `lbd`.
- They get a Dockerfile, a `.dockerignore`, and a report.
- They run `docker build` and get an image in the 15–30 MB range.
- They run `lbd --validate` and it confirms the image works.
- They run `lbd doctor` and get useful diagnostics if anything is suspicious.

Everything that doesn't contribute to this end state is not part of the MVP.

---

## Tracking progress

Phase status is tracked in `ROADMAP.md` (separate document, lighter weight). This document — `DEVELOPMENT_PLAN.md` — is the **plan**. `ROADMAP.md` is the **current state**.

When a phase completes, mark it in `ROADMAP.md` and review this plan for any adjustments based on what was learned. The plan is allowed to evolve; that's why it lives in version control.
