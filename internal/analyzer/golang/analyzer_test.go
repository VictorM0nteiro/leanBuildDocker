package golang_test

import (
	"path/filepath"
	"testing"

	goanalyzer "github.com/VictorM0nteiro/leanBuildDocker/internal/analyzer/golang"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "..", "testdata", "projects", name)
}

func TestGoAnalyzer_SimpleAPI(t *testing.T) {
	inv, _ := scanner.Scan(fixture("simple-api"))
	info, err := goanalyzer.New().Analyze(inv)
	if err != nil {
		t.Fatal(err)
	}

	if info.ModulePath != "example.com/simple-api" {
		t.Errorf("ModulePath = %q, want example.com/simple-api", info.ModulePath)
	}
	if info.LanguageVersion == "" {
		t.Error("LanguageVersion is empty")
	}
	if info.HasCGO {
		t.Error("HasCGO = true, want false")
	}
	if info.UsesVendoring {
		t.Error("UsesVendoring = true, want false")
	}
	if len(info.MainPackages) != 1 {
		t.Fatalf("len(MainPackages) = %d, want 1", len(info.MainPackages))
	}
	if info.EntryPoint != "./cmd/api" {
		t.Errorf("EntryPoint = %q, want ./cmd/api", info.EntryPoint)
	}
	if len(info.DirectDependencies) == 0 {
		t.Error("DirectDependencies is empty")
	}
}

func TestGoAnalyzer_CGO(t *testing.T) {
	inv, _ := scanner.Scan(fixture("cgo-sqlite"))
	info, err := goanalyzer.New().Analyze(inv)
	if err != nil {
		t.Fatal(err)
	}
	if !info.HasCGO {
		t.Error("HasCGO = false, want true")
	}
}

func TestGoAnalyzer_Monorepo(t *testing.T) {
	inv, _ := scanner.Scan(fixture("monorepo"))
	info, err := goanalyzer.New().Analyze(inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.MainPackages) != 2 {
		t.Errorf("len(MainPackages) = %d, want 2", len(info.MainPackages))
	}
	if info.EntryPoint != "" {
		t.Errorf("EntryPoint = %q, want empty for monorepo", info.EntryPoint)
	}
}
