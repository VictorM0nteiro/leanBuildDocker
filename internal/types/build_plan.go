package types

// BuildPlan represents the decisions a Planner made about how to build
// a project, based on a ProjectInfo. The Decisions slice is the direct
// source of truth for .lbd-report.md.
type BuildPlan struct {
	BuildStage   BuildStage
	RuntimeStage RuntimeStage

	BinaryName string
	BinaryPath string

	UseMultiStage bool

	Decisions []Decision
}

type BuildStage struct {
	BaseImage      string
	SystemPackages []string
	PackageManager string // "apk" | "apt" — empty when no packages
	BuildCommand   []string
	CopyVendor     bool
}

type RuntimeStage struct {
	BaseImage      string
	SystemPackages []string
	PackageManager string
	EntryCommand   []string
	User           string
	WorkingDir     string
	ExposedPorts   []int
}

// PackageSet groups system package names by distribution. The Planner
// chooses one slice based on the build/runtime base image.
type PackageSet struct {
	Alpine []string `yaml:"alpine"`
	Debian []string `yaml:"debian"`
}

type Decision struct {
	Topic        string
	Chose        string
	Because      string
	Alternatives []string
}
