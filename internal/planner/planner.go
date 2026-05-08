package planner

import (
	"fmt"
	"path"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// MonorepoError is returned when the Planner cannot pick a single entry
// point because the project has multiple cmd/* binaries and no --target.
type MonorepoError struct {
	MainPackages []string
}

func (e *MonorepoError) Error() string {
	return fmt.Sprintf(
		"Detected: Go monorepo (%d cmd/* binaries found)\nCannot infer which service to containerize.\nRun: lbd --target ./cmd/<name>",
		len(e.MainPackages),
	)
}

// Plan converts a ProjectInfo into a BuildPlan by applying Go-specific rules.
func Plan(info *types.ProjectInfo) (*types.BuildPlan, error) {
	if info.EntryPoint == "" {
		if len(info.MainPackages) > 1 {
			return nil, &MonorepoError{MainPackages: info.MainPackages}
		}
		return nil, fmt.Errorf("no entry point: project has no main.go in root or cmd/*")
	}

	plan := &types.BuildPlan{UseMultiStage: true}
	plan.BinaryName = binaryNameFor(info.EntryPoint)
	plan.BinaryPath = "/" + plan.BinaryName

	chooseImages(info, plan)
	chooseBuildCommand(info, plan)
	resolveSystemPackages(info, plan)
	chooseRuntimeConfig(plan)

	return plan, nil
}

func binaryNameFor(entryPoint string) string {
	clean := strings.TrimPrefix(entryPoint, "./")
	if clean == "" || clean == "." {
		return "app"
	}
	return path.Base(clean)
}

func chooseImages(info *types.ProjectInfo, plan *types.BuildPlan) {
	goVer := info.LanguageVersion
	if goVer == "" {
		goVer = "1.23"
	}

	if info.HasCGO {
		plan.BuildStage.BaseImage = "golang:" + goVer
		plan.RuntimeStage.BaseImage = "gcr.io/distroless/base-debian12:nonroot"
		plan.Decisions = append(plan.Decisions,
			types.Decision{
				Topic:        "build base image",
				Chose:        plan.BuildStage.BaseImage,
				Because:      "CGO is enabled and requires gcc + glibc, available in the full Debian Go image",
				Alternatives: []string{"golang:" + goVer + "-alpine (would need musl-dev, alpine-sdk)"},
			},
			types.Decision{
				Topic:        "runtime base image",
				Chose:        plan.RuntimeStage.BaseImage,
				Because:      "CGO produces dynamically-linked binaries that need glibc at runtime",
				Alternatives: []string{"distroless/static (would crash without libc)", "alpine (smaller, but uses musl)"},
			},
		)
		return
	}

	plan.BuildStage.BaseImage = "golang:" + goVer + "-alpine"
	plan.RuntimeStage.BaseImage = "gcr.io/distroless/static-debian12:nonroot"
	plan.Decisions = append(plan.Decisions,
		types.Decision{
			Topic:        "build base image",
			Chose:        plan.BuildStage.BaseImage,
			Because:      "no CGO detected; Alpine-based Go image is smaller and sufficient for static builds",
			Alternatives: []string{"golang:" + goVer + " (Debian, larger)"},
		},
		types.Decision{
			Topic:        "runtime base image",
			Chose:        plan.RuntimeStage.BaseImage,
			Because:      "static Go binary needs no OS — distroless/static gives the smallest secure runtime",
			Alternatives: []string{"scratch (no certs, no /etc/passwd)", "alpine (~5MB extra for shell utilities)"},
		},
	)
}

func chooseBuildCommand(info *types.ProjectInfo, plan *types.BuildPlan) {
	args := []string{"go", "build"}

	if !info.HasCGO {
		args = append(args, "-ldflags=-w -s", "-trimpath")
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "build flags",
			Chose:   "-ldflags=-w -s -trimpath",
			Because: "strips debug info and removes filesystem paths, reducing binary size and improving reproducibility",
		})
	}

	if info.UsesVendoring {
		args = append(args, "-mod=vendor")
		plan.BuildStage.CopyVendor = true
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "dependency mode",
			Chose:   "-mod=vendor",
			Because: "vendor/ directory is present, so dependencies ship with the source",
		})
	}

	args = append(args, "-o", "/out/"+plan.BinaryName, info.EntryPoint)
	plan.BuildStage.BuildCommand = args
}

func chooseRuntimeConfig(plan *types.BuildPlan) {
	plan.RuntimeStage.User = "nonroot"
	plan.RuntimeStage.WorkingDir = "/"
	plan.RuntimeStage.ExposedPorts = []int{8080}
	plan.RuntimeStage.EntryCommand = []string{plan.BinaryPath}

	plan.Decisions = append(plan.Decisions,
		types.Decision{
			Topic:        "runtime user",
			Chose:        "nonroot (UID 65532)",
			Because:      "running as non-root is a security best practice; built into distroless images",
			Alternatives: []string{"root (insecure)"},
		},
		types.Decision{
			Topic:   "exposed port",
			Chose:   "8080",
			Because: "default for Go web services; not inferred from source (Phase 3 limitation)",
		},
	)
}

func resolveSystemPackages(info *types.ProjectInfo, plan *types.BuildPlan) {
	if len(info.SystemPackageHints) == 0 {
		return
	}

	isAlpineBuild := strings.Contains(plan.BuildStage.BaseImage, "alpine")
	if isAlpineBuild {
		plan.BuildStage.PackageManager = "apk"
	} else {
		plan.BuildStage.PackageManager = "apt"
	}

	var buildPkgs []string
	for _, hint := range info.SystemPackageHints {
		if isAlpineBuild {
			buildPkgs = append(buildPkgs, hint.Build.Alpine...)
		} else {
			buildPkgs = append(buildPkgs, hint.Build.Debian...)
		}
	}
	plan.BuildStage.SystemPackages = buildPkgs

	if len(buildPkgs) > 0 {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "build-time system packages",
			Chose:   strings.Join(buildPkgs, ", "),
			Because: "required by Go dependencies that link against C libraries (sourced from leanBuildDocker knowledge base)",
		})
	}
}

