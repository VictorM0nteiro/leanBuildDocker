package analyzer

import (
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

type dummyAnalyzer struct{}

func NewDummy() Analyzer { return &dummyAnalyzer{} }

func (d *dummyAnalyzer) Language() string { return "go" }

func (d *dummyAnalyzer) Analyze(inv *scanner.FileInventory) (*types.ProjectInfo, error) {
	return &types.ProjectInfo{
		Language:        "go",
		LanguageVersion: "1.23",
		ModulePath:      "example.com/dummy",
		ProjectRoot:     inv.ProjectRoot,
		DirectDependencies: []types.GoDep{
			{Path: "github.com/gin-gonic/gin", Version: "v1.9.1"},
		},
		EntryPoint: "./cmd/api",
		HasCGO:     false,
	}, nil
}
