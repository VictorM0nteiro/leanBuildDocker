package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// webPlan mirrors what the planner produces for a layered web service with
// runtime assets, framework env, version stamping, and cache mounts.
func webPlan() *types.BuildPlan {
	return &types.BuildPlan{
		BinaryName:     "server",
		BinaryPath:     "/app/server",
		UseMultiStage:  true,
		UseCacheMounts: true,
		BuildStage: types.BuildStage{
			BaseImage:  "golang:1.23-alpine",
			VersionArg: "dev",
			BuildCommand: []string{
				"go", "build", "-ldflags=-w -s -X main.version=$VERSION",
				"-trimpath", "-o", "/out/server", "./cmd/server",
			},
		},
		RuntimeStage: types.RuntimeStage{
			BaseImage:    "gcr.io/distroless/static-debian12:nonroot",
			EntryCommand: []string{"/app/server"},
			User:         "nonroot",
			WorkingDir:   "/app",
			ExposedPorts: []int{9090},
			Env:          []types.EnvVar{{Key: "GIN_MODE", Value: "release"}, {Key: "PORT", Value: "9090"}},
			Assets: []types.AssetCopy{
				{Src: "templates", Dst: "/app/templates"},
				{Src: "config", Dst: "/app/config"},
			},
		},
		Decisions: []types.Decision{{Topic: "t", Chose: "c", Because: "b"}},
	}
}

func TestRender_DeepDockerfile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(webPlan(), tmpDir, Options{}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, "Dockerfile"))
	df := string(data)

	wants := []string{
		"# syntax=docker/dockerfile:1",
		"--mount=type=cache,target=/go/pkg/mod",
		"ARG VERSION=dev",
		`"-ldflags=-w -s -X main.version=$VERSION"`, // quoted so it stays one shell token
		"WORKDIR /app",
		"COPY --from=builder /out/server /app/server",
		"COPY --from=builder /src/templates /app/templates",
		"COPY --from=builder /src/config /app/config",
		"ENV GIN_MODE=release",
		"ENV PORT=9090",
		"EXPOSE 9090",
		`ENTRYPOINT ["/app/server"]`,
	}
	for _, w := range wants {
		if !strings.Contains(df, w) {
			t.Errorf("Dockerfile missing %q\n--- got ---\n%s", w, df)
		}
	}
}

func TestRender_ReportHasSummaryAndDecisions(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(webPlan(), tmpDir, Options{}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, ".lbd-report.md"))
	report := string(data)

	for _, w := range []string{"## Summary", "gcr.io/distroless/static-debian12:nonroot", "## Decisions"} {
		if !strings.Contains(report, w) {
			t.Errorf("report missing %q", w)
		}
	}
}
