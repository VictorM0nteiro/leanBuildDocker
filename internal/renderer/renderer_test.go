package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

func samplePlan() *types.BuildPlan {
	return &types.BuildPlan{
		BinaryName:    "api",
		BinaryPath:    "/api",
		UseMultiStage: true,
		BuildStage: types.BuildStage{
			BaseImage:    "golang:1.23-alpine",
			BuildCommand: []string{"go", "build", "-ldflags=-w -s", "-trimpath", "-o", "/out/api", "./cmd/api"},
		},
		RuntimeStage: types.RuntimeStage{
			BaseImage:    "gcr.io/distroless/static-debian12:nonroot",
			EntryCommand: []string{"/api"},
			User:         "nonroot",
			WorkingDir:   "/",
			ExposedPorts: []int{8080},
		},
		Decisions: []types.Decision{
			{
				Topic:        "runtime base image",
				Chose:        "gcr.io/distroless/static-debian12:nonroot",
				Because:      "static Go binary needs no OS",
				Alternatives: []string{"scratch", "alpine"},
			},
		},
	}
}

func TestRender_CreatesExpectedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	if err := Render(samplePlan(), tmpDir); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, name := range []string{"Dockerfile", ".lbd-report.md"} {
		path := filepath.Join(tmpDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestRender_DockerfileContent(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(samplePlan(), tmpDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	df := string(data)

	wants := []string{
		"FROM golang:1.23-alpine AS builder",
		"FROM gcr.io/distroless/static-debian12:nonroot",
		"COPY --from=builder /out/api /api",
		"USER nonroot",
		"EXPOSE 8080",
		`ENTRYPOINT ["/api"]`,
	}
	for _, w := range wants {
		if !strings.Contains(df, w) {
			t.Errorf("Dockerfile missing %q\n--- got ---\n%s", w, df)
		}
	}
}

func TestRender_ReportListsDecisions(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(samplePlan(), tmpDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".lbd-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	report := string(data)

	if !strings.Contains(report, "runtime base image") {
		t.Errorf("report missing decision topic, got:\n%s", report)
	}
	if !strings.Contains(report, "Alternatives considered") {
		t.Errorf("report missing alternatives section, got:\n%s", report)
	}
}

func TestRender_FailsOnNonexistentDir(t *testing.T) {
	err := Render(samplePlan(), "/this/path/does/not/exist/at/all")
	if err == nil {
		t.Error("expected Render to fail on nonexistent directory, got nil error")
	}
}
