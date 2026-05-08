package planner_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/planner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

func TestPlan_NoCGONoVendor(t *testing.T) {
	info := &types.ProjectInfo{
		LanguageVersion: "1.23",
		EntryPoint:      "./cmd/api",
		MainPackages:    []string{"cmd/api"},
	}
	plan, err := planner.Plan(info)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.BuildStage.BaseImage, "alpine") {
		t.Errorf("BuildStage.BaseImage = %q, want alpine variant", plan.BuildStage.BaseImage)
	}
	if !strings.Contains(plan.RuntimeStage.BaseImage, "static") {
		t.Errorf("RuntimeStage.BaseImage = %q, want distroless/static", plan.RuntimeStage.BaseImage)
	}
	if plan.BinaryName != "api" {
		t.Errorf("BinaryName = %q, want api", plan.BinaryName)
	}
	if plan.RuntimeStage.User != "nonroot" {
		t.Errorf("User = %q, want nonroot", plan.RuntimeStage.User)
	}
	if len(plan.Decisions) == 0 {
		t.Error("expected decisions to be recorded")
	}
}

func TestPlan_CGO(t *testing.T) {
	info := &types.ProjectInfo{
		LanguageVersion: "1.23",
		EntryPoint:      ".",
		MainPackages:    []string{"."},
		HasCGO:          true,
	}
	plan, err := planner.Plan(info)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.BuildStage.BaseImage, "alpine") {
		t.Errorf("CGO must use Debian build base, got %q", plan.BuildStage.BaseImage)
	}
	if strings.Contains(plan.RuntimeStage.BaseImage, "static") {
		t.Errorf("CGO must use distroless/base, got %q", plan.RuntimeStage.BaseImage)
	}
	if plan.BinaryName != "app" {
		t.Errorf("BinaryName = %q, want app (root entry point)", plan.BinaryName)
	}
}

func TestPlan_Vendoring(t *testing.T) {
	info := &types.ProjectInfo{
		LanguageVersion: "1.23",
		EntryPoint:      "./cmd/api",
		MainPackages:    []string{"cmd/api"},
		UsesVendoring:   true,
	}
	plan, err := planner.Plan(info)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.BuildStage.CopyVendor {
		t.Error("expected CopyVendor=true with vendoring")
	}
	found := false
	for _, a := range plan.BuildStage.BuildCommand {
		if a == "-mod=vendor" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected -mod=vendor in build command, got %v", plan.BuildStage.BuildCommand)
	}
}

func TestPlan_MonorepoFails(t *testing.T) {
	info := &types.ProjectInfo{
		LanguageVersion: "1.23",
		MainPackages:    []string{"cmd/api", "cmd/worker"},
	}
	_, err := planner.Plan(info)
	if err == nil {
		t.Fatal("expected error for monorepo without target")
	}

	var monoErr *planner.MonorepoError
	if !errors.As(err, &monoErr) {
		t.Errorf("expected MonorepoError, got %T", err)
	}
	if !strings.Contains(err.Error(), "Run: lbd --target") {
		t.Errorf("error must suggest --target, got: %s", err.Error())
	}
}
