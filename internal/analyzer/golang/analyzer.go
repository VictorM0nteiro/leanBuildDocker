package golang

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// GoAnalyzer implements analyzer.Analyzer for Go projects.
type GoAnalyzer struct{}

func New() *GoAnalyzer { return &GoAnalyzer{} }

func (g *GoAnalyzer) Language() string { return "go" }

func (g *GoAnalyzer) Analyze(inv *scanner.FileInventory) (*types.ProjectInfo, error) {
	info := &types.ProjectInfo{
		Language:    "go",
		ProjectRoot: inv.ProjectRoot,
	}

	if err := g.parseModFile(inv.ProjectRoot, info); err != nil {
		return nil, err
	}

	info.UsesVendoring = g.hasVendorDir(inv.ProjectRoot)

	mains, err := g.findMainPackages(inv.ProjectRoot)
	if err != nil {
		return nil, err
	}
	info.MainPackages = mains
	info.EntryPoint = selectEntryPoint(mains)

	hasCGO, err := g.detectCGO(inv.ProjectRoot)
	if err != nil {
		return nil, err
	}
	info.HasCGO = hasCGO

	return info, nil
}

func (g *GoAnalyzer) parseModFile(root string, info *types.ProjectInfo) error {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	info.ModulePath = f.Module.Mod.Path
	if f.Go != nil {
		info.LanguageVersion = f.Go.Version
	}

	for _, req := range f.Require {
		info.DirectDependencies = append(info.DirectDependencies, types.GoDep{
			Path:     req.Mod.Path,
			Version:  req.Mod.Version,
			Indirect: req.Indirect,
		})
	}

	return nil
}

func (g *GoAnalyzer) hasVendorDir(root string) bool {
	fi, err := os.Stat(filepath.Join(root, "vendor"))
	return err == nil && fi.IsDir()
}

func (g *GoAnalyzer) findMainPackages(root string) ([]string, error) {
	cmdDir := filepath.Join(root, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if errors.Is(err, os.ErrNotExist) {
		if _, err2 := os.Stat(filepath.Join(root, "main.go")); err2 == nil {
			return []string{"."}, nil
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading cmd/: %w", err)
	}

	var mains []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(cmdDir, e.Name(), "main.go")); err == nil {
			// filepath.ToSlash garante forward slashes em Windows (Go package paths)
			mains = append(mains, filepath.ToSlash(filepath.Join("cmd", e.Name())))
		}
	}
	return mains, nil
}

func selectEntryPoint(mains []string) string {
	if len(mains) == 1 {
		return "./" + mains[0]
	}
	return ""
}

func (g *GoAnalyzer) detectCGO(root string) (bool, error) {
	fset := token.NewFileSet()
	found := false

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		for _, imp := range f.Imports {
			if imp.Path.Value == `"C"` {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	return found, err
}
