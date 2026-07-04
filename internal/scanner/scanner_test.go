package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTree materializes a set of relative paths (files) under a temp dir.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func TestScan_FindsNestedFilesAndDirs(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":                          "module x",
		"cmd/server/main.go":              "package main",
		"internal/service/deep/deep.go":   "package deep",
		".git/config":                     "[core]",
		"node_modules/left-pad/index.js":  "module.exports = 0",
	})

	inv, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"go.mod", "cmd/server/main.go", "internal/service/deep/deep.go"} {
		if !contains(inv.Files, f) {
			t.Errorf("expected file %q in inventory, got %v", f, inv.Files)
		}
	}
	for _, d := range []string{"cmd", "cmd/server", "internal/service/deep"} {
		if !contains(inv.Dirs, d) {
			t.Errorf("expected dir %q in inventory, got %v", d, inv.Dirs)
		}
	}
}

func TestScan_PrunesVCSAndDependencyDirs(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":                         "module x",
		".git/config":                    "[core]",
		"node_modules/pkg/index.js":      "0",
		".idea/workspace.xml":            "<x/>",
	})

	inv, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range inv.Files {
		if wantsPrune(f) {
			t.Errorf("pruned path leaked into inventory: %q", f)
		}
	}
	for _, d := range inv.Dirs {
		if wantsPrune(d) {
			t.Errorf("pruned dir leaked into inventory: %q", d)
		}
	}
}

func wantsPrune(p string) bool {
	for _, bad := range []string{".git", "node_modules", ".idea"} {
		if p == bad || len(p) > len(bad) && p[:len(bad)+1] == bad+"/" {
			return true
		}
	}
	return false
}

func TestInventory_Helpers(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":             "module x",
		"web/templates/a.html": "<a/>",
	})
	inv, _ := Scan(root)

	if !inv.HasFile("go.mod") {
		t.Error("HasFile(go.mod) = false")
	}
	if inv.HasFile("go.work") {
		t.Error("HasFile(go.work) = true, want false")
	}
	if !inv.HasDir("templates") {
		t.Error("HasDir(templates) = false")
	}
	if got := inv.Abs("go.mod"); got != filepath.Join(root, "go.mod") {
		t.Errorf("Abs(go.mod) = %q", got)
	}
}
