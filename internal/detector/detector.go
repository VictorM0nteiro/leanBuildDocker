// Package detector identifies a project's primary language from a scanned
// file inventory. In MVP the only supported language is Go, but detection is
// evidence-based (marker files + source presence) rather than assumed, so an
// unsupported project fails loudly instead of being mis-analyzed.
package detector

import (
	"fmt"
	"strings"

	scanner "github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

// Result represents what the Detector concluded about a project.
type Result struct {
	Language   string
	Confidence float64
	// Evidence lists the concrete signals that drove the decision, for
	// verbose logging and the report.
	Evidence []string
}

// Detect identifies the project's primary language from a FileInventory.
//
// Confidence is graded, not binary:
//   - go.mod present            → 1.0 (canonical module marker)
//   - go.work present, no mod   → 0.9 (workspace root)
//   - only .go files, no go.mod → 0.5 (loose source; buildable but unusual)
//
// A project with no Go evidence yields a clear "unsupported" error rather
// than being pushed through the Go analyzer.
func Detect(inv *scanner.FileInventory) (*Result, error) {
	var evidence []string
	hasGoMod := inv.HasFile("go.mod")
	hasGoWork := inv.HasFile("go.work")

	goFiles := 0
	for _, f := range inv.Files {
		if strings.HasSuffix(f, ".go") {
			goFiles++
		}
	}

	switch {
	case hasGoMod:
		evidence = append(evidence, "go.mod")
		if hasGoWork {
			evidence = append(evidence, "go.work")
		}
		evidence = append(evidence, fmt.Sprintf("%d .go files", goFiles))
		return &Result{Language: "go", Confidence: 1.0, Evidence: evidence}, nil

	case hasGoWork:
		return &Result{
			Language:   "go",
			Confidence: 0.9,
			Evidence:   []string{"go.work", fmt.Sprintf("%d .go files", goFiles)},
		}, nil

	case goFiles > 0:
		return &Result{
			Language:   "go",
			Confidence: 0.5,
			Evidence:   []string{fmt.Sprintf("%d .go files, no go.mod", goFiles)},
		}, nil

	default:
		return nil, fmt.Errorf(
			"no supported language detected: found no go.mod, go.work, or .go files in %s\nleanBuildDocker MVP supports Go projects only",
			inv.ProjectRoot,
		)
	}
}
