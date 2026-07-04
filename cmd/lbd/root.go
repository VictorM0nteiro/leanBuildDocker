package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/analyzer/golang"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/detector"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/logging"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/planner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/renderer"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/validator"
)

// resolveProjectRoot finds the module root by walking up from the current
// working directory looking for go.mod, so `lbd` works from any subdirectory.
// The --target flag selects a service *within* this root; it does not change
// where the root is (that was a bug — a monorepo's cmd/api has no go.mod).
func resolveProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if dir != wd {
				slog.Info("found go.mod above cwd", "root", dir)
			}
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}

// relTarget converts a --target path (possibly relative to the cwd or
// absolute) into a package path relative to the module root, which is the
// form the analyzer expects for entry-point selection.
func relTarget(root, target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolving --target %q: %w", target, err)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("--target %q is outside the project root %q", target, root)
	}
	return filepath.ToSlash(rel), nil
}

var version = "0.0.0-dev"

type rootFlags struct {
	target   string
	verbose  bool
	force    bool
	validate bool
	keep     bool
}

func newRootCmd() *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:     "lbd",
		Short:   "Generate optimized Dockerfiles from Go projects",
		Long:    "leanBuildDocker analyzes a Go project and generates a production-grade Dockerfile, .dockerignore, and explanatory report.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			logging.Configure(flags.verbose)
			return runPipeline(flags)
		},
	}

	cmd.Flags().StringVarP(&flags.target, "target", "t", "", "path to the project (default: current directory)")
	cmd.Flags().BoolVarP(&flags.verbose, "verbose", "v", false, "enable verbose output")
	cmd.Flags().BoolVarP(&flags.force, "force", "f", false, "overwrite existing Dockerfile, .dockerignore, and .lbd-report.md")
	cmd.Flags().BoolVar(&flags.validate, "validate", false, "after generating, build the image and run a smoke test")
	cmd.Flags().BoolVar(&flags.keep, "keep", false, "keep the validation image instead of removing it")

	cmd.AddCommand(newDoctorCmd())

	return cmd
}

func runPipeline(flags *rootFlags) error {
	projectPath, err := resolveProjectRoot()
	if err != nil {
		return err
	}

	stat, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target directory does not exist: %s", projectPath)
		}
		return fmt.Errorf("checking target: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("target is not a directory: %s", projectPath)
	}

	slog.Debug("starting pipeline", "target", projectPath)

	inv, err := scanner.Scan(projectPath)
	if err != nil {
		return fmt.Errorf("scanning project: %w", err)
	}
	slog.Debug("scanner finished", "files", len(inv.Files))

	det, err := detector.Detect(inv)
	if err != nil {
		return fmt.Errorf("detecting language: %w", err)
	}
	slog.Info("language detected", "language", det.Language, "confidence", det.Confidence, "evidence", det.Evidence)

	var opts []golang.Option
	if flags.target != "" {
		rel, err := relTarget(projectPath, flags.target)
		if err != nil {
			return err
		}
		opts = append(opts, golang.WithTarget(rel))
	}

	info, err := golang.New(opts...).Analyze(inv)
	if err != nil {
		return fmt.Errorf("analyzing project: %w", err)
	}
	slog.Debug("analyzer finished",
		"dependencies", len(info.DirectDependencies),
		"cgo", info.HasCGO,
		"entry", info.EntryPoint,
		"framework", info.Framework,
		"ports", info.ExposedPorts,
		"runtime_assets", len(info.RuntimeAssets),
		"needs_ca_certs", info.NeedsCACerts,
		"needs_tzdata", info.NeedsTZData,
	)
	for _, w := range info.Warnings {
		slog.Warn("analysis warning", "message", w.Message)
	}

	plan, err := planner.Plan(info)
	if err != nil {
		return fmt.Errorf("planning build: %w", err)
	}
	slog.Debug("planner finished", "build_image", plan.BuildStage.BaseImage, "runtime_image", plan.RuntimeStage.BaseImage)

	if err := renderer.Render(plan, projectPath, renderer.Options{Force: flags.force}); err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}
	slog.Info("output written", "files", []string{"Dockerfile", ".lbd-report.md"})

	if flags.validate {
		res := validator.Validate(validator.Options{
			ProjectDir: projectPath,
			Keep:       flags.keep,
		})
		printValidationReport(os.Stdout, res)
		if !res.Build.OK || (res.Build.OK && !res.Smoke.OK) {
			return fmt.Errorf("validation failed")
		}
	}

	return nil
}
