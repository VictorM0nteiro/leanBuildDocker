// internal/analyzer/dummy.go
package analyzer

import (
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// dummyAnalyzer is a Phase 1 placeholder. Will be deleted in Phase 2
// once the real Go analyzer (in subpackage golang/) is implemented.
type dummyAnalyzer struct{}

func NewDummy() Analyzer {
	return &dummyAnalyzer{}
}

func (d *dummyAnalyzer) Language() string { return "go" }

func (d *dummyAnalyzer) Analyze(inv *scanner.FileInventory) (*types.ProjectInfo, error) {
	return &types.ProjectInfo{
		Language:        "go",
		LanguageVersion: "1.23",
		Dependencies:    []string{"github.com/gin-gonic/gin"},
		EntryPoint:      "./cmd/api",
		HasCGO:          false,
	}, nil
}