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

	// UseCacheMounts enables BuildKit cache mounts for the module and
	// build caches. Requires DOCKER_BUILDKIT=1 (default on modern Docker).
	UseCacheMounts bool

	// IsWorkspace mirrors ProjectInfo.IsWorkspace so the template can copy
	// go.work and skip the single-module go.mod pre-copy optimization.
	IsWorkspace bool

	Decisions []Decision
}

type BuildStage struct {
	BaseImage      string
	SystemPackages []string
	PackageManager string // "apk" | "apt" — empty when no packages
	BuildCommand   []string
	CopyVendor     bool

	// VersionArg, when set, is injected as an ARG and stamped into the
	// binary via -ldflags -X <VersionVar>=$VersionArg.
	VersionArg string
}

type RuntimeStage struct {
	BaseImage      string
	SystemPackages []string
	PackageManager string
	EntryCommand   []string
	User           string
	WorkingDir     string
	ExposedPorts   []int

	// Env holds environment variables to set in the runtime image, in
	// declaration order (e.g. GIN_MODE=release).
	Env []EnvVar

	// Assets are files/directories copied from the build context into the
	// runtime image so the program can read them at runtime.
	Assets []AssetCopy

	// HealthCheck, when non-nil, renders a HEALTHCHECK instruction.
	HealthCheck *HealthCheck
}

// EnvVar is a single runtime environment variable.
type EnvVar struct {
	Key   string
	Value string
}

// AssetCopy describes a COPY from the build context (Src, relative to the
// project root) to the runtime image (Dst, absolute path).
type AssetCopy struct {
	Src string
	Dst string
}

// HealthCheck renders a Docker HEALTHCHECK instruction.
type HealthCheck struct {
	Test     []string
	Interval string
	Timeout  string
	Retries  int
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
