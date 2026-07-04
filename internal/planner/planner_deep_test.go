package planner_test

import (
	"strings"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/planner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

func webInfo() *types.ProjectInfo {
	return &types.ProjectInfo{
		LanguageVersion: "1.23",
		EntryPoint:      "./cmd/server",
		MainPackages:    []string{"cmd/server"},
		Framework:       "gin",
		ExposedPorts:    []int{9090},
		PortFromEnv:     true,
		NeedsCACerts:    true,
		NeedsTZData:     true,
		VersionVar:      "main.version",
		RuntimeAssets: []types.RuntimeAsset{
			{Path: "templates", Kind: "html templates", IsDir: true},
			{Path: "config", Kind: "config files", IsDir: true},
		},
	}
}

func hasAssetCopy(plan *types.BuildPlan, src, dst string) bool {
	for _, a := range plan.RuntimeStage.Assets {
		if a.Src == src && a.Dst == dst {
			return true
		}
	}
	return false
}

func hasEnv(plan *types.BuildPlan, key, value string) bool {
	for _, e := range plan.RuntimeStage.Env {
		if e.Key == key && e.Value == value {
			return true
		}
	}
	return false
}

func TestPlan_AssetsMoveUnderApp(t *testing.T) {
	plan, err := planner.Plan(webInfo())
	if err != nil {
		t.Fatal(err)
	}

	if plan.RuntimeStage.WorkingDir != "/app" {
		t.Errorf("WorkingDir = %q, want /app", plan.RuntimeStage.WorkingDir)
	}
	if plan.BinaryPath != "/app/server" {
		t.Errorf("BinaryPath = %q, want /app/server", plan.BinaryPath)
	}
	if !hasAssetCopy(plan, "templates", "/app/templates") {
		t.Errorf("missing templates asset copy, got %v", plan.RuntimeStage.Assets)
	}
	if !hasAssetCopy(plan, "config", "/app/config") {
		t.Errorf("missing config asset copy, got %v", plan.RuntimeStage.Assets)
	}
}

func TestPlan_PortAndFrameworkEnv(t *testing.T) {
	plan, err := planner.Plan(webInfo())
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.RuntimeStage.ExposedPorts) != 1 || plan.RuntimeStage.ExposedPorts[0] != 9090 {
		t.Errorf("ExposedPorts = %v, want [9090]", plan.RuntimeStage.ExposedPorts)
	}
	if !hasEnv(plan, "GIN_MODE", "release") {
		t.Errorf("missing GIN_MODE=release, got %v", plan.RuntimeStage.Env)
	}
	if !hasEnv(plan, "PORT", "9090") {
		t.Errorf("missing PORT=9090 (port comes from env), got %v", plan.RuntimeStage.Env)
	}
}

func TestPlan_VersionStamping(t *testing.T) {
	plan, err := planner.Plan(webInfo())
	if err != nil {
		t.Fatal(err)
	}
	if plan.BuildStage.VersionArg != "dev" {
		t.Errorf("VersionArg = %q, want dev", plan.BuildStage.VersionArg)
	}
	joined := strings.Join(plan.BuildStage.BuildCommand, " ")
	if !strings.Contains(joined, "-X main.version=$VERSION") {
		t.Errorf("build command missing version ldflag, got %v", plan.BuildStage.BuildCommand)
	}
}

func TestPlan_DefaultsPortWhenUnknown(t *testing.T) {
	info := &types.ProjectInfo{
		LanguageVersion: "1.23",
		EntryPoint:      "./cmd/api",
		MainPackages:    []string{"cmd/api"},
	}
	plan, err := planner.Plan(info)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.RuntimeStage.ExposedPorts) != 1 || plan.RuntimeStage.ExposedPorts[0] != 8080 {
		t.Errorf("ExposedPorts = %v, want [8080] default", plan.RuntimeStage.ExposedPorts)
	}
	// No assets: binary stays at root, WORKDIR /.
	if plan.BinaryPath != "/api" {
		t.Errorf("BinaryPath = %q, want /api", plan.BinaryPath)
	}
}

func TestPlan_CacheMountsEnabled(t *testing.T) {
	plan, err := planner.Plan(webInfo())
	if err != nil {
		t.Fatal(err)
	}
	if !plan.UseCacheMounts {
		t.Error("UseCacheMounts = false, want true")
	}
}
