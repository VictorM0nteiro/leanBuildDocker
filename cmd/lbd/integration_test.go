package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPipeline_WritesArtifactsToTarget(t *testing.T) {
	// Use a temp directory as the "project" path.
	// Phase 1 uses dummy data, so the directory contents don't matter —
	// we only care that artifacts are written to it.
	tmpDir := t.TempDir()

	flags := &rootFlags{
		target:  tmpDir,
		verbose: false,
	}

	if err := runPipeline(flags); err != nil {
		t.Fatalf("runPipeline failed: %v", err)
	}

	// Check that the expected output files exist.
	for _, name := range []string{"Dockerfile", ".lbd-report.md"} {
		path := filepath.Join(tmpDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected %s to have content, got empty file", name)
		}
	}
}