package types

import "testing"

func TestProjectInfo_ZeroValueIsUsable(t *testing.T) {
	// Zero values should be safe to use without panic.
	// This protects against future field additions that might
	// require explicit initialization (e.g., maps, channels).
	var info ProjectInfo
	if info.Language != "" {
		t.Errorf("expected empty Language, got %q", info.Language)
	}
	if info.HasCGO {
		t.Errorf("expected HasCGO=false by default")
	}
	if len(info.Dependencies) != 0 {
		t.Errorf("expected empty Dependencies, got %d items", len(info.Dependencies))
	}
}