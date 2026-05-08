package knowledge_test

import (
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/knowledge"
)

func TestLoad_FindsSqliteEntry(t *testing.T) {
	kb, err := knowledge.Load()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := kb.Lookup("github.com/mattn/go-sqlite3")
	if !ok {
		t.Fatal("expected sqlite entry to be present")
	}
	if !entry.RequiresCGO {
		t.Error("sqlite entry should be marked as requiring CGO")
	}
	if len(entry.Build.Alpine) == 0 || len(entry.Build.Debian) == 0 {
		t.Errorf("sqlite must have build packages on both distros, got alpine=%v debian=%v",
			entry.Build.Alpine, entry.Build.Debian)
	}
}

func TestLookup_UnknownReturnsFalse(t *testing.T) {
	kb, err := knowledge.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := kb.Lookup("github.com/does/not/exist"); ok {
		t.Error("expected unknown path to return false")
	}
}
