package planner


import (
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// Plan converts a ProjectInfo into a BuildPlan.
// Phase 1 stub: returns a hardcoded plan regardless of input.
func Plan(info *types.ProjectInfo) (*types.BuildPlan, error) {
	return &types.BuildPlan{
		BuildBaseImage:   "golang:1.23-alpine",
		RuntimeBaseImage: "gcr.io/distroless/static-debian12:nonroot",
		BuildCommand:     []string{"go", "build", "-o", "/out/api", "./cmd/api"},
		RunCommand:       []string{"/api"},
		ExposedPort:      8080,
		UseMultiStage:    true,
	}, nil
}