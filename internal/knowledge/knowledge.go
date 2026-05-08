package knowledge

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

//go:embed packages.yaml
var packagesYAML []byte

// Entry is a single record in the knowledge base.
type Entry struct {
	GoPath      string            `yaml:"go_path"`
	RequiresCGO bool              `yaml:"requires_cgo"`
	Build       types.PackageSet  `yaml:"build"`
	Runtime     types.PackageSet  `yaml:"runtime"`
}

// KnowledgeBase is an in-memory, indexed view of packages.yaml.
type KnowledgeBase struct {
	entries map[string]Entry
}

type kbFile struct {
	Packages []Entry `yaml:"packages"`
}

// Load parses the embedded YAML and returns an indexed KnowledgeBase.
func Load() (*KnowledgeBase, error) {
	var f kbFile
	if err := yaml.Unmarshal(packagesYAML, &f); err != nil {
		return nil, fmt.Errorf("parsing knowledge base: %w", err)
	}

	entries := make(map[string]Entry, len(f.Packages))
	for _, e := range f.Packages {
		entries[e.GoPath] = e
	}
	return &KnowledgeBase{entries: entries}, nil
}

// Lookup returns the entry for a Go dependency path, if any.
func (kb *KnowledgeBase) Lookup(goPath string) (Entry, bool) {
	e, ok := kb.entries[goPath]
	return e, ok
}
