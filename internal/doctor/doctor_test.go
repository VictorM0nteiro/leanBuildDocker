package doctor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/doctor"
)

func setupProject(t *testing.T, withCGO bool) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, dir, "go.mod", "module example.com/x\n\ngo 1.23\n")
	main := "package main\n\nfunc main() {}\n"
	if withCGO {
		main = "package main\n\n// #include <stdlib.h>\nimport \"C\"\nfunc main() {}\n"
	}
	mustWrite(t, dir, "main.go", main)
	return dir
}

func mustWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hasTopic(r *doctor.Report, topic string) bool {
	for _, f := range r.Findings {
		if f.Topic == topic {
			return true
		}
	}
	return false
}

func hasSeverity(r *doctor.Report, s doctor.Severity) bool {
	for _, f := range r.Findings {
		if f.Severity == s {
			return true
		}
	}
	return false
}

func TestDiagnose_NoDockerfile(t *testing.T) {
	dir := setupProject(t, false)
	rep, err := doctor.Diagnose(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTopic(rep, "Dockerfile") {
		t.Errorf("expected info finding for missing Dockerfile, got %+v", rep.Findings)
	}
}

func TestDiagnose_CGOWithStaticRuntime(t *testing.T) {
	dir := setupProject(t, true)
	mustWrite(t, dir, "Dockerfile",
		"FROM golang:1.23 AS b\nFROM gcr.io/distroless/static-debian12:nonroot\nUSER nonroot\nCMD [\"/app\"]\n")

	rep, err := doctor.Diagnose(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSeverity(rep, doctor.SeverityError) {
		t.Errorf("expected error severity for CGO + distroless/static, got %+v", rep.Findings)
	}
}

func TestDiagnose_RootUser(t *testing.T) {
	dir := setupProject(t, false)
	mustWrite(t, dir, "Dockerfile", "FROM alpine\nCMD [\"/app\"]\n")

	rep, err := doctor.Diagnose(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTopic(rep, "runtime user") {
		t.Errorf("expected runtime user finding, got %+v", rep.Findings)
	}
}

func TestDiagnose_CleanProject(t *testing.T) {
	dir := setupProject(t, false)
	mustWrite(t, dir, "Dockerfile",
		"FROM golang:1.23-alpine AS b\nFROM gcr.io/distroless/static-debian12:nonroot\nUSER nonroot\nEXPOSE 8080\nENTRYPOINT [\"/app\"]\n")

	rep, err := doctor.Diagnose(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.Severity == doctor.SeverityError || f.Severity == doctor.SeverityWarn {
			t.Errorf("clean project should produce no warnings/errors, got: %+v", f)
		}
	}
}

func TestDiagnose_TestsInBuildContext(t *testing.T) {
	dir := setupProject(t, false)
	mustWrite(t, dir, "main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n")

	rep, err := doctor.Diagnose(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasTopic(rep, ".dockerignore") {
		t.Errorf("expected .dockerignore finding, got %+v", rep.Findings)
	}
}
