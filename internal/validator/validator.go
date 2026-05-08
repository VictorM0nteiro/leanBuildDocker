package validator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Options configures a validation run.
type Options struct {
	ProjectDir string // directory containing the generated Dockerfile
	Keep       bool   // don't delete the validation image after running
}

// Result is the outcome of a validation run, suitable for printing.
type Result struct {
	ImageTag       string
	Build          StageResult
	ImageSizeBytes int64

	Smoke SmokeResult

	HadolintRan    bool
	HadolintIssues []string
	HadolintErr    error

	Cleaned bool
}

type StageResult struct {
	OK     bool
	Output string
	Err    error
}

type SmokeResult struct {
	OK       bool
	ExitCode int
	Logs     string
	Findings []Finding
}

// Validate builds the Dockerfile, runs a smoke test, lints with hadolint,
// and (by default) removes the validation image. It never panics on
// missing tools; instead, the corresponding fields stay zero / false.
func Validate(opts Options) *Result {
	res := &Result{ImageTag: makeTag()}

	res.Build = dockerBuild(opts.ProjectDir, res.ImageTag)
	if !res.Build.OK {
		return res
	}

	if size, err := dockerImageSize(res.ImageTag); err == nil {
		res.ImageSizeBytes = size
	}

	res.Smoke = smokeTest(res.ImageTag)
	res.HadolintRan, res.HadolintIssues, res.HadolintErr = runHadolint(opts.ProjectDir)

	if !opts.Keep {
		if err := exec.Command("docker", "rmi", "-f", res.ImageTag).Run(); err == nil {
			res.Cleaned = true
		}
	}
	return res
}

func makeTag() string {
	return fmt.Sprintf("lbd-validate-%d", time.Now().UnixNano())
}

func dockerBuild(dir, tag string) StageResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var combined bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", tag, dir)
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err := cmd.Run()
	return StageResult{OK: err == nil, Output: combined.String(), Err: err}
}

func dockerImageSize(tag string) (int64, error) {
	out, err := exec.Command("docker", "image", "inspect", tag, "--format", "{{.Size}}").Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

// smokeTest starts the container detached, waits 5s, inspects state,
// and grabs logs. The container is removed at the end of the call.
func smokeTest(tag string) SmokeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runOut, err := exec.CommandContext(ctx, "docker", "run", "-d", tag).Output()
	if err != nil {
		return SmokeResult{Logs: extractStderr(err)}
	}
	id := strings.TrimSpace(string(runOut))
	defer func() { _ = exec.Command("docker", "rm", "-f", id).Run() }()

	time.Sleep(5 * time.Second)

	stateOut, _ := exec.Command("docker", "inspect", "--format", "{{.State.ExitCode}}|{{.State.Running}}", id).Output()
	parts := strings.SplitN(strings.TrimSpace(string(stateOut)), "|", 2)
	exitCode, _ := strconv.Atoi(parts[0])
	running := len(parts) > 1 && parts[1] == "true"

	logsOut, _ := exec.Command("docker", "logs", id).CombinedOutput()

	res := SmokeResult{ExitCode: exitCode, Logs: string(logsOut)}
	res.OK = running || exitCode == 0
	if !res.OK {
		res.Findings = AnalyzeStderr(res.Logs)
	}
	return res
}

func runHadolint(dir string) (bool, []string, error) {
	if _, err := exec.LookPath("hadolint"); err != nil {
		return false, nil, nil
	}
	out, err := exec.Command("hadolint", filepath.Join(dir, "Dockerfile")).CombinedOutput()

	var issues []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			issues = append(issues, line)
		}
	}

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		// real failure, not just "found warnings"
		return true, issues, err
	}
	return true, issues, nil
}

func extractStderr(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(exitErr.Stderr)
	}
	return err.Error()
}
