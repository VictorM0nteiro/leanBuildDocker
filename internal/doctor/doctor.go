package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/analyzer/golang"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// Severity classifies a Finding. Doctor exits non-zero only on Error.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Finding is one issue Doctor identified.
type Finding struct {
	Severity Severity
	Topic    string
	Message  string
	Suggest  string
}

// Report bundles all findings plus enough context to render them.
type Report struct {
	ProjectDir       string
	DockerfilePath   string
	DockerfileExists bool
	Findings         []Finding
}

// Diagnose runs all checks against projectDir and the Dockerfile inside it.
// Missing or unreadable artifacts produce findings rather than errors —
// the doctor's job is to be useful even on partial inputs.
func Diagnose(projectDir string) (*Report, error) {
	rep := &Report{
		ProjectDir:     projectDir,
		DockerfilePath: filepath.Join(projectDir, "Dockerfile"),
	}

	inv, err := scanner.Scan(projectDir)
	if err != nil {
		return nil, fmt.Errorf("scanning project: %w", err)
	}

	var info *types.ProjectInfo
	if i, err := golang.New().Analyze(inv); err == nil {
		info = i
	} else {
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityWarn,
			Topic:    "project analysis",
			Message:  "could not analyze Go project: " + err.Error(),
		})
	}

	dfBytes, dfErr := os.ReadFile(rep.DockerfilePath)
	rep.DockerfileExists = dfErr == nil
	df := string(dfBytes)

	if !rep.DockerfileExists {
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityInfo,
			Topic:    "Dockerfile",
			Message:  "no Dockerfile found in project root",
			Suggest:  "run `lbd` to generate one",
		})
	}

	if rep.DockerfileExists && info != nil {
		rep.Findings = append(rep.Findings, checkCGOMismatch(info, df)...)
		rep.Findings = append(rep.Findings, checkRuntimeUser(df)...)
		rep.Findings = append(rep.Findings, checkCACerts(projectDir, df)...)
		rep.Findings = append(rep.Findings, checkExpose(df)...)
	}
	rep.Findings = append(rep.Findings, checkDockerignore(projectDir)...)

	return rep, nil
}
