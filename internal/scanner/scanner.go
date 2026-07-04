// Package scanner walks a project directory and produces a flat inventory
// of its files and directories. It is deliberately language-agnostic: it
// applies only coarse ignore rules (VCS metadata, dependency caches, editor
// junk) and leaves all interpretation to the Detector and Analyzer.
package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// FileInventory is the result of scanning a project directory. Paths are
// relative to ProjectRoot and use forward slashes on every platform so the
// rest of the pipeline can treat them as Go package/URL-style paths.
type FileInventory struct {
	ProjectRoot string
	Files       []string // relative file paths (forward slash)
	Dirs        []string // relative directory paths (forward slash)
}

// prunedDirs are directory names we never descend into. They either hold
// no source we care about (VCS, editor state) or are dependency caches the
// Analyzer inspects by presence alone (vendor).
var prunedDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	".idea":        true,
	".vscode":      true,
	"node_modules": true,
	".lbd-cache":   true,
}

// Scan walks projectPath recursively and returns its inventory. The walk is
// deterministic: filepath.WalkDir yields entries in lexical order, so the
// resulting slices are stable across runs — a property the report and tests
// both rely on.
func Scan(projectPath string) (*FileInventory, error) {
	inv := &FileInventory{ProjectRoot: projectPath}

	err := filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(projectPath, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if prunedDirs[d.Name()] {
				return filepath.SkipDir
			}
			inv.Dirs = append(inv.Dirs, rel)
			return nil
		}

		inv.Files = append(inv.Files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", projectPath, err)
	}

	return inv, nil
}

// HasFile reports whether a file with the given base name exists anywhere in
// the inventory. Useful for cheap marker checks (go.mod, go.work, Makefile).
func (inv *FileInventory) HasFile(base string) bool {
	for _, f := range inv.Files {
		if pathBase(f) == base {
			return true
		}
	}
	return false
}

// HasDir reports whether a directory with the given base name exists anywhere
// in the inventory.
func (inv *FileInventory) HasDir(base string) bool {
	for _, dir := range inv.Dirs {
		if pathBase(dir) == base {
			return true
		}
	}
	return false
}

// Abs joins a relative inventory path back onto the project root, returning an
// OS-native absolute path suitable for os.ReadFile.
func (inv *FileInventory) Abs(rel string) string {
	return filepath.Join(inv.ProjectRoot, filepath.FromSlash(rel))
}

func pathBase(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
