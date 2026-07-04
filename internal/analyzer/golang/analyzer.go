// Package golang implements the Go language Analyzer. It reads go.mod /
// go.work, walks the project's source with the Go parser, and produces a
// types.ProjectInfo describing real, observed facts about the project —
// entry points at any depth, web framework, listen ports, runtime assets,
// CGO, build tags, and CA/timezone needs. No code is executed.
package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/knowledge"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// GoAnalyzer implements analyzer.Analyzer for Go projects.
type GoAnalyzer struct {
	target string // explicit entry-point package selected via --target
}

// Option configures a GoAnalyzer.
type Option func(*GoAnalyzer)

// WithTarget selects a specific main package as the entry point. It is how the
// CLI resolves a monorepo: the user points --target at one cmd/* package.
func WithTarget(target string) Option {
	return func(g *GoAnalyzer) { g.target = target }
}

func New(opts ...Option) *GoAnalyzer {
	g := &GoAnalyzer{}
	for _, o := range opts {
		o(g)
	}
	return g
}

func (g *GoAnalyzer) Language() string { return "go" }

// Analyze folds every source signal into a single ProjectInfo.
func (g *GoAnalyzer) Analyze(inv *scanner.FileInventory) (*types.ProjectInfo, error) {
	info := &types.ProjectInfo{
		Language:    "go",
		ProjectRoot: inv.ProjectRoot,
	}

	if err := g.parseModFile(inv.ProjectRoot, info); err != nil {
		return nil, err
	}
	g.parseWorkFile(inv, info)

	info.UsesVendoring = g.hasVendorDir(inv.ProjectRoot)

	facts := scanSource(inv)

	info.HasCGO = facts.hasCGO
	info.MainPackages = normalizeMains(facts.mainDirs)
	entry, warn := g.resolveEntryPoint(info.MainPackages)
	info.EntryPoint = entry
	if warn != "" {
		info.Warnings = append(info.Warnings, types.Warning{Message: warn})
	}

	info.Framework = pickFramework(facts.frameworks)
	info.ExposedPorts = sortedPorts(facts.ports)
	info.PortFromEnv = facts.portFromEnv
	info.NeedsCACerts = facts.needsCerts
	// An explicit `import _ "time/tzdata"` embeds the zoneinfo DB into the
	// binary, so the runtime image needs nothing extra.
	info.NeedsTZData = facts.needsTZ && !facts.imports["time/tzdata"]
	info.BuildTags = sortedKeys(facts.buildTags)
	info.VersionVar = facts.versionVar

	assets, embedded, assetWarnings := detectAssets(inv, facts)
	info.RuntimeAssets = assets
	info.EmbeddedAssets = embedded
	info.Warnings = append(info.Warnings, assetWarnings...)

	g.resolveSystemPackages(facts, info)

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

// parseWorkFile records multi-module workspace facts when a go.work exists.
// Failure to parse is non-fatal: the project still builds as a single module.
func (g *GoAnalyzer) parseWorkFile(inv *scanner.FileInventory, info *types.ProjectInfo) {
	if !inv.HasFile("go.work") {
		return
	}
	data, err := os.ReadFile(filepath.Join(inv.ProjectRoot, "go.work"))
	if err != nil {
		return
	}
	wf, err := modfile.ParseWork("go.work", data, nil)
	if err != nil {
		return
	}
	info.IsWorkspace = true
	for _, use := range wf.Use {
		info.WorkspaceModules = append(info.WorkspaceModules, filepath.ToSlash(use.Path))
	}
	sort.Strings(info.WorkspaceModules)
}

func (g *GoAnalyzer) hasVendorDir(root string) bool {
	fi, err := os.Stat(filepath.Join(root, "vendor"))
	return err == nil && fi.IsDir()
}

// resolveEntryPoint picks the package to containerize. An explicit --target
// wins; otherwise a single discovered main is used, and multiple mains leave
// the entry point empty for the Planner to reject with a monorepo error.
func (g *GoAnalyzer) resolveEntryPoint(mains []string) (entry, warning string) {
	if g.target != "" {
		ep := normalizeEntry(g.target)
		if !containsEntry(mains, ep) && len(mains) > 0 {
			warning = fmt.Sprintf("--target %s is not among the discovered main packages %v; using it anyway", ep, mainsAsEntries(mains))
		}
		return ep, warning
	}
	return selectEntryPoint(mains), ""
}

// resolveSystemPackages consults the knowledge base for every module the
// project depends on (direct and indirect) and every import actually used in
// source, unioning the hits. Using both catches C-linking deps that appear
// only as indirect requirements.
func (g *GoAnalyzer) resolveSystemPackages(facts *sourceFacts, info *types.ProjectInfo) {
	kb, err := knowledge.Load()
	if err != nil {
		info.Warnings = append(info.Warnings, types.Warning{Message: "knowledge base unavailable: " + err.Error()})
		return
	}

	seen := map[string]bool{}
	add := func(p string) {
		if seen[p] {
			return
		}
		if entry, ok := kb.Lookup(p); ok {
			seen[p] = true
			info.SystemPackageHints = append(info.SystemPackageHints, types.SystemPackageHint{
				DepPath: entry.GoPath,
				Build:   entry.Build,
				Runtime: entry.Runtime,
			})
		}
	}

	for _, dep := range info.DirectDependencies {
		add(dep.Path)
	}
	for imp := range facts.imports {
		add(imp)
	}

	sort.Slice(info.SystemPackageHints, func(i, j int) bool {
		return info.SystemPackageHints[i].DepPath < info.SystemPackageHints[j].DepPath
	})
}

// --- entry-point helpers ---

// normalizeMains turns AST-discovered directories ("." or "cmd/api") into the
// canonical package-path form stored in ProjectInfo.MainPackages.
func normalizeMains(dirs []string) []string {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, filepath.ToSlash(d))
	}
	sort.Strings(out)
	return out
}

func selectEntryPoint(mains []string) string {
	if len(mains) == 1 {
		return toEntry(mains[0])
	}
	return ""
}

// toEntry renders a main package dir as a `go build` argument: "." for the
// module root, "./cmd/api" for a subdirectory.
func toEntry(dir string) string {
	if dir == "." || dir == "" {
		return "."
	}
	return "./" + strings.TrimPrefix(filepath.ToSlash(dir), "./")
}

func normalizeEntry(target string) string {
	t := filepath.ToSlash(strings.TrimSpace(target))
	t = strings.TrimSuffix(t, "/")
	if t == "" || t == "." {
		return "."
	}
	if strings.HasPrefix(t, "./") {
		return t
	}
	return "./" + t
}

func containsEntry(mains []string, entry string) bool {
	for _, m := range mains {
		if toEntry(m) == entry {
			return true
		}
	}
	return false
}

func mainsAsEntries(mains []string) []string {
	out := make([]string, len(mains))
	for i, m := range mains {
		out[i] = toEntry(m)
	}
	return out
}

// --- fact-folding helpers ---

func pickFramework(set map[string]bool) string {
	for _, fw := range frameworkByPrefix {
		if set[fw.name] {
			return fw.name
		}
	}
	// net/http as a fallback server signal is handled by the planner via
	// framework=="" plus port detection; we don't label bare net/http here.
	return ""
}

func sortedPorts(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
