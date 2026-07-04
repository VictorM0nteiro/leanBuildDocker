package detector_test

import (
	"testing"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/detector"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

func TestDetect_GoModIsFullConfidence(t *testing.T) {
	inv := &scanner.FileInventory{
		ProjectRoot: "/x",
		Files:       []string{"go.mod", "main.go"},
	}
	res, err := detector.Detect(inv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Language != "go" || res.Confidence != 1.0 {
		t.Errorf("got %+v, want go@1.0", res)
	}
}

func TestDetect_GoWorkOnly(t *testing.T) {
	inv := &scanner.FileInventory{
		ProjectRoot: "/x",
		Files:       []string{"go.work", "svc/main.go"},
	}
	res, err := detector.Detect(inv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Confidence != 0.9 {
		t.Errorf("Confidence = %v, want 0.9", res.Confidence)
	}
}

func TestDetect_LooseGoFiles(t *testing.T) {
	inv := &scanner.FileInventory{
		ProjectRoot: "/x",
		Files:       []string{"main.go"},
	}
	res, err := detector.Detect(inv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Confidence != 0.5 {
		t.Errorf("Confidence = %v, want 0.5", res.Confidence)
	}
}

func TestDetect_NoGoFailsLoudly(t *testing.T) {
	inv := &scanner.FileInventory{
		ProjectRoot: "/x",
		Files:       []string{"package.json", "index.js"},
	}
	if _, err := detector.Detect(inv); err == nil {
		t.Fatal("expected error for a non-Go project")
	}
}
