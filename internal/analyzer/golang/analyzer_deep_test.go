package golang_test

import (
	"testing"

	goanalyzer "github.com/VictorM0nteiro/leanBuildDocker/internal/analyzer/golang"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

func analyze(t *testing.T, name string, opts ...goanalyzer.Option) *types.ProjectInfo {
	t.Helper()
	inv, err := scanner.Scan(fixture(name))
	if err != nil {
		t.Fatal(err)
	}
	info, err := goanalyzer.New(opts...).Analyze(inv)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func hasAsset(info *types.ProjectInfo, path string) bool {
	for _, a := range info.RuntimeAssets {
		if a.Path == path {
			return true
		}
	}
	return false
}

// TestDeepNested proves the core fix: a main package buried several levels
// deep (services/api/cmd) — which the old scanner could not find — is now
// discovered and selected as the entry point.
func TestDeepNested_FindsBuriedMain(t *testing.T) {
	info := analyze(t, "deep-nested")

	if len(info.MainPackages) != 1 {
		t.Fatalf("MainPackages = %v, want exactly one", info.MainPackages)
	}
	if info.EntryPoint != "./services/api/cmd" {
		t.Errorf("EntryPoint = %q, want ./services/api/cmd", info.EntryPoint)
	}
}

func TestLayeredWeb_RealAnalysis(t *testing.T) {
	info := analyze(t, "layered-web")

	if info.EntryPoint != "./cmd/server" {
		t.Errorf("EntryPoint = %q, want ./cmd/server", info.EntryPoint)
	}
	if info.Framework != "gin" {
		t.Errorf("Framework = %q, want gin", info.Framework)
	}
	if len(info.ExposedPorts) != 1 || info.ExposedPorts[0] != 9090 {
		t.Errorf("ExposedPorts = %v, want [9090]", info.ExposedPorts)
	}
	if !info.PortFromEnv {
		t.Error("PortFromEnv = false, want true (os.Getenv(\"PORT\"))")
	}
	if !info.NeedsCACerts {
		t.Error("NeedsCACerts = false, want true (outbound http.Get)")
	}
	if !info.NeedsTZData {
		t.Error("NeedsTZData = false, want true (time.LoadLocation)")
	}
	if info.VersionVar != "main.version" {
		t.Errorf("VersionVar = %q, want main.version", info.VersionVar)
	}

	for _, want := range []string{"templates", "static", "migrations", "config"} {
		if !hasAsset(info, want) {
			t.Errorf("expected runtime asset %q, got %v", want, info.RuntimeAssets)
		}
	}
}

// TestEmbeddedAssets proves the //go:embed judgement: templates/ is baked into
// the binary, so it must NOT be a runtime asset to copy.
func TestEmbeddedAssets_NotCopied(t *testing.T) {
	info := analyze(t, "embedded-assets")

	if hasAsset(info, "templates") {
		t.Error("templates/ is embedded via go:embed; it must not be a copied runtime asset")
	}
	if len(info.EmbeddedAssets) == 0 {
		t.Error("expected templates/ to be recorded as an embedded asset")
	}
	if len(info.ExposedPorts) != 1 || info.ExposedPorts[0] != 8000 {
		t.Errorf("ExposedPorts = %v, want [8000]", info.ExposedPorts)
	}
	// A bare net/http *server* (ListenAndServe) does not imply outbound TLS.
	if info.NeedsCACerts {
		t.Error("NeedsCACerts = true; a server with no outbound client should not need CA certs")
	}
}

func TestTarget_SelectsEntryPointInMonorepo(t *testing.T) {
	info := analyze(t, "monorepo", goanalyzer.WithTarget("cmd/worker"))

	if info.EntryPoint != "./cmd/worker" {
		t.Errorf("EntryPoint = %q, want ./cmd/worker", info.EntryPoint)
	}
}
