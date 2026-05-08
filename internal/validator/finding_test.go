package validator_test

import (
	"strings"
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/validator"
)

func TestAnalyzeStderr_NoSuchFile(t *testing.T) {
	fs := validator.AnalyzeStderr("/api: no such file or directory")
	if len(fs) == 0 {
		t.Fatal("expected a finding")
	}
	if !strings.Contains(fs[0].Suggest, "distroless") {
		t.Errorf("suggestion should mention distroless, got: %s", fs[0].Suggest)
	}
}

func TestAnalyzeStderr_SharedObject(t *testing.T) {
	fs := validator.AnalyzeStderr("loading shared libraries: libfoo.so: cannot open shared object file")
	if len(fs) == 0 {
		t.Fatal("expected a finding")
	}
	if !strings.Contains(fs[0].Suggest, "CGO_ENABLED=0") {
		t.Errorf("suggestion should mention CGO, got: %s", fs[0].Suggest)
	}
}

func TestAnalyzeStderr_ExecFormat(t *testing.T) {
	fs := validator.AnalyzeStderr("standard_init_linux.go: exec format error")
	if len(fs) == 0 || !strings.Contains(fs[0].Suggest, "architecture") {
		t.Errorf("expected architecture finding, got: %v", fs)
	}
}

func TestAnalyzeStderr_NoMatch(t *testing.T) {
	if fs := validator.AnalyzeStderr("starting api on :8080"); len(fs) != 0 {
		t.Errorf("expected no findings, got %d", len(fs))
	}
}
