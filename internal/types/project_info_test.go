package types

import "testing"

func TestProjectInfo_ZeroValueIsUsable(t *testing.T) {
	var info ProjectInfo
	if info.Language != "" {
		t.Errorf("expected empty Language, got %q", info.Language)
	}
	if info.HasCGO {
		t.Errorf("expected HasCGO=false by default")
	}
	if len(info.DirectDependencies) != 0 {
		t.Errorf("expected empty DirectDependencies, got %d items", len(info.DirectDependencies))
	}
}
