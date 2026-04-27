package detector

import (
	scanner "github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

// Result represents what the Detector concluded about a project.
type Result struct {
	Language string
	Confidence float64
}

// Detect identifies the project's primary language from a FileInventory.
// Phase 1 stub: always returns Go with full confidence.
func Detect(inv *scanner.FileInvetory) (*Result, error) {
	return &Result{
		Language: "go",
		Confidence: 1.0,
	}, nil
} 