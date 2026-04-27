package types

// BuildPlan represents the decisions a Planner made about how to build
// a project, based on a ProjectInfo.
type BuildPlan struct {
	BuildBaseImage string
	
	RuntimeBaseImage string

	// BuildCommand is the command that compiles the project, in Docker's
	// exec form (a slice of arguments, not a single shell string).
	// Exec form avoids spawning a shell and gives proper signal handling.
	// Example: ["go", "build", "-o", "/out/api", "./cmd/api"]
	BuildCommand []string

	RunCommand []string

	ExposedPort int

	// UseMultiStage controls whether the Dockerfile is rendered with
	// separate build and runtime stages. In practice this should always
	// be true for Go (multi-stage is what makes images small), but the
	// flag exists so the Renderer can branch cleanly without inferring it.
	UseMultiStage bool
}