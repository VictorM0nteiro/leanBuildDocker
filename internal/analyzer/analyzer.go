package analyzer

import (
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// Analyzer is the interface every language-specific analyzer implements.
// In Phase 1 we have a single dummy implementation; Phase 2 introduces
// the real Go analyzer in the golang/ subpackage.
type Analyzer interface {
	Language() string
	Analyze(inv *scanner.FileInventory) (*types.ProjectInfo, error)
}