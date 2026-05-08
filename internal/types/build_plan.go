package types

// BuildPlan represents the decisions a Planner made about how to build
// a project, based on a ProjectInfo. The Decisions slice is the direct
// source of truth for .lbd-report.md.
type BuildPlan struct {
	BuildStage   BuildStage
	RuntimeStage RuntimeStage

	BinaryName string // e.g. "api"
	BinaryPath string // path inside runtime image, e.g. "/api"

	UseMultiStage bool

	Decisions []Decision
}

// BuildStage carries the configuration of the first (compile) stage.
type BuildStage struct {
	BaseImage      string
	SystemPackages []string // build-time apk/apt packages
	BuildCommand   []string // exec form: ["go", "build", "-o", "/out/api", "./cmd/api"]
	CopyVendor     bool     // when vendoring is used, copy vendor/ into /src
}

// RuntimeStage carries the configuration of the final image.
type RuntimeStage struct {
	BaseImage      string
	SystemPackages []string // runtime-only packages (e.g., ca-certificates)
	EntryCommand   []string // exec form: ["/api"]
	User           string   // "nonroot" or "65532:65532"
	WorkingDir     string
	ExposedPorts   []int
}

// Decision is a non-trivial choice the Planner made, recorded for the report.
type Decision struct {
	Topic        string
	Chose        string
	Because      string
	Alternatives []string
}
