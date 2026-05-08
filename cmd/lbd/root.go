package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/analyzer"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/detector"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/logging"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/planner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/renderer"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

var version = "0.0.0-dev"

type rootFlags struct {
	target  string
	verbose bool
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

	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newAnalyzeCmd())
	cmd.AddCommand(newExplainCmd())

	return cmd
}

func runPipeline(flags *rootFlags) error {
	projectPath := flags.target
	if projectPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		projectPath = wd
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
	slog.Info("language detected", "language", det.Language, "confidence", det.Confidence)

	a := analyzer.NewDummy()
	info, err := a.Analyze(inv)
	if err != nil {
		return fmt.Errorf("analyzing project: %w", err)
	}
	slog.Debug("analyzer finished", "dependencies", len(info.DirectDependencies), "cgo", info.HasCGO)

	plan, err := planner.Plan(info)
	if err != nil {
		return fmt.Errorf("planning build: %w", err)
	}
	slog.Debug("planner finished", "build_image", plan.BuildBaseImage, "runtime_image", plan.RuntimeBaseImage)

	if err := renderer.Render(plan, projectPath); err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}
	slog.Info("output written", "files", []string{"Dockerfile", ".lbd-report.md"})

	return nil
}