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

func TestRender_CreatesAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(samplePlan(), tmpDir, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Dockerfile", ".dockerignore", ".lbd-report.md"} {
		info, err := os.Stat(filepath.Join(tmpDir, name))
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestRender_DockerfileContent(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Render(samplePlan(), tmpDir, Options{}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, "Dockerfile"))
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

func TestRender_RefusesToOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("hand-written"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Render(samplePlan(), tmpDir, Options{Force: false})
	if err == nil {
		t.Fatal("expected Render to refuse overwrite, got nil error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "Dockerfile"))
	if string(data) != "hand-written" {
		t.Errorf("existing file was clobbered: %q", data)
	}
}

func TestRender_ForceOverwrites(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("old"), 0644)

	if err := Render(samplePlan(), tmpDir, Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, "Dockerfile"))
	if !strings.Contains(string(data), "FROM golang") {
		t.Errorf("expected new content, got: %s", data)
	}
}

func TestRender_AtomicOnTemplateFailure(t *testing.T) {
	// A nil plan should fail mid-render. Ensure no partial files are written.
	tmpDir := t.TempDir()
	err := Render(nil, tmpDir, Options{})
	if err == nil {
		t.Fatal("expected failure with nil plan")
	}
	for _, name := range []string{"Dockerfile", ".dockerignore", ".lbd-report.md"} {
		if _, statErr := os.Stat(filepath.Join(tmpDir, name)); statErr == nil {
			t.Errorf("partial file written: %s", name)
		}
	}
}

func TestRender_FailsOnNonexistentDir(t *testing.T) {
	err := Render(samplePlan(), "/this/path/does/not/exist/at/all", Options{Force: true})
	if err == nil {
		t.Error("expected failure on nonexistent directory")
	}
}
