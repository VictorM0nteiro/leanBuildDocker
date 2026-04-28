package renderer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// Render writes the output artifacts (Dockerfile, .dockerignore, report)
// to outputDir based on the BuildPlan.
// Phase 1 stub: writes hardcoded content, ignores the plan.
func Render(plan *types.BuildPlan, outputDir string) error {
	dockerfile := "# Phase 1 dummy Dockerfile\nFROM scratch\n"
	report := "# leanBuildDocker Report\n\nThis is a placeholder report.\n"

	if err := writeFile(outputDir, "Dockerfile", dockerfile); err != nil {
		return err
	}
	if err := writeFile(outputDir, ".lbd-report.md", report); err != nil {
		return err
	}
	return nil
}

func writeFile(dir, name, content string) error {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}