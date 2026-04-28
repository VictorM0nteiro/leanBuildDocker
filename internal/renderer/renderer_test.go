package renderer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

func TestRender_CreatesExpectedFiles(t *testing.T) {
	// t.TempDir() gives a fresh temp directory automatically cleaned up
	// after the test. No leftover files in /tmp, no manual cleanup.
	tmpDir := t.TempDir()

	plan := &types.BuildPlan{
		BuildBaseImage:   "golang:1.23-alpine",
		RuntimeBaseImage: "gcr.io/distroless/static-debian12:nonroot",
		BuildCommand:     []string{"go", "build", "-o", "/out/api", "."},
		RunCommand:       []string{"/api"},
		ExposedPort:      8080,
		UseMultiStage:    true,
	}

	if err := Render(plan, tmpDir); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	// Verify both expected files exist.
	for _, name := range []string{"Dockerfile", ".lbd-report.md"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
		}
	}
}

func TestRender_FailsOnNonexistentDir(t *testing.T) {
	plan := &types.BuildPlan{} // empty plan is fine for this test

	err := Render(plan, "/this/path/does/not/exist/at/all")
	if err == nil {
		t.Error("expected Render to fail on nonexistent directory, got nil error")
	}
}