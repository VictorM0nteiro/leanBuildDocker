// Package planner turns the observed facts in a types.ProjectInfo into the
// concrete build decisions of a types.BuildPlan. It is language-agnostic in
// spirit — it reads ProjectInfo fields, never Go ASTs — and records a
// Decision for every non-obvious choice so the report can explain itself.
package planner

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// MonorepoError is returned when the Planner cannot pick a single entry
// point because the project has multiple main packages and no --target.
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
		return nil, fmt.Errorf("no entry point: found no package main with a func main in the project")
	}

	plan := &types.BuildPlan{
		UseMultiStage:  true,
		UseCacheMounts: true,
		IsWorkspace:    info.IsWorkspace,
	}
	plan.BinaryName = binaryNameFor(info.EntryPoint)

	chooseImages(info, plan)
	chooseBuildCommand(info, plan)
	resolveSystemPackages(info, plan)
	chooseRuntimeLayout(info, plan)
	chooseUser(plan)
	choosePorts(info, plan)
	chooseFrameworkEnv(info, plan)
	noteRuntimeNeeds(info, plan)
	noteWorkspace(info, plan)
	noteBuildTags(info, plan)
	noteCacheMounts(plan)

	return plan, nil
}

func binaryNameFor(entryPoint string) string {
	clean := strings.TrimPrefix(entryPoint, "./")
	if clean == "" || clean == "." {
		return "app"
	}
	base := path.Base(clean)
	// A generic leaf like ".../cmd" or ".../server/main" names nothing useful;
	// the parent directory is the real service name (e.g. services/api/cmd → api).
	if (base == "cmd" || base == "main") && path.Dir(clean) != "." {
		if parent := path.Base(path.Dir(clean)); parent != "." && parent != "" {
			return parent
		}
	}
	return base
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
				Because:      "CGO is enabled (an import \"C\" was found), which requires gcc + glibc, available in the full Debian Go image",
				Alternatives: []string{"golang:" + goVer + "-alpine (would need musl-dev, alpine-sdk)"},
			},
			types.Decision{
				Topic:        "runtime base image",
				Chose:        plan.RuntimeStage.BaseImage,
				Because:      "CGO produces dynamically-linked binaries that need glibc at runtime; distroless/base bundles glibc, CA certs, and tzdata",
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
			Because:      "no CGO detected; the Alpine-based Go image is smaller and sufficient for static builds",
			Alternatives: []string{"golang:" + goVer + " (Debian, larger)"},
		},
		types.Decision{
			Topic:        "runtime base image",
			Chose:        plan.RuntimeStage.BaseImage,
			Because:      "the static Go binary needs no OS — distroless/static gives the smallest secure runtime, and still bundles CA certs, tzdata, and a nonroot user",
			Alternatives: []string{"scratch (no certs, no /etc/passwd)", "alpine (~5MB extra for shell utilities)"},
		},
	)
}

func chooseBuildCommand(info *types.ProjectInfo, plan *types.BuildPlan) {
	args := []string{"go", "build"}

	var ldflags []string
	if !info.HasCGO {
		ldflags = append(ldflags, "-w", "-s")
	}
	if info.VersionVar != "" {
		plan.BuildStage.VersionArg = "dev"
		ldflags = append(ldflags, "-X", info.VersionVar+"=$VERSION")
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "version stamping",
			Chose:   "-ldflags -X " + info.VersionVar + "=$VERSION (ARG VERSION)",
			Because: "a package-level version variable was found; the build stamps it from a --build-arg VERSION so images are traceable to a release",
		})
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags="+strings.Join(ldflags, " "))
	}
	if !info.HasCGO {
		args = append(args, "-trimpath")
	}

	if len(ldflags) > 0 || !info.HasCGO {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "build flags",
			Chose:   strings.TrimSpace(strings.Join(args[2:], " ")),
			Because: "strips debug info and filesystem paths, shrinking the binary and making builds reproducible",
		})
	}

	if info.UsesVendoring {
		args = append(args, "-mod=vendor")
		plan.BuildStage.CopyVendor = true
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "dependency mode",
			Chose:   "-mod=vendor",
			Because: "a vendor/ directory is present, so dependencies ship with the source and no network fetch is needed",
		})
	}

	args = append(args, "-o", "/out/"+plan.BinaryName, info.EntryPoint)
	plan.BuildStage.BuildCommand = args
}

// chooseRuntimeLayout decides where the binary and any runtime assets live in
// the final image. When the program reads assets from disk, everything moves
// under /app and WORKDIR is set there so relative paths (templates/, etc.)
// resolve exactly as they do in development.
func chooseRuntimeLayout(info *types.ProjectInfo, plan *types.BuildPlan) {
	if len(info.RuntimeAssets) == 0 {
		plan.RuntimeStage.WorkingDir = "/"
		plan.BinaryPath = "/" + plan.BinaryName
		plan.RuntimeStage.EntryCommand = []string{plan.BinaryPath}
		return
	}

	plan.RuntimeStage.WorkingDir = "/app"
	plan.BinaryPath = "/app/" + plan.BinaryName
	plan.RuntimeStage.EntryCommand = []string{plan.BinaryPath}

	var kinds []string
	for _, a := range info.RuntimeAssets {
		plan.RuntimeStage.Assets = append(plan.RuntimeStage.Assets, types.AssetCopy{
			Src: a.Path,
			Dst: "/app/" + a.Path,
		})
		kinds = append(kinds, a.Path+" ("+a.Kind+")")
	}
	plan.Decisions = append(plan.Decisions, types.Decision{
		Topic:        "runtime assets",
		Chose:        "COPY " + strings.Join(kinds, ", ") + " into /app",
		Because:      "these paths are read from disk at runtime and are not embedded via //go:embed, so they must ship in the image; WORKDIR /app keeps relative paths working",
		Alternatives: []string{"embed them with //go:embed to bake them into the binary and drop these COPY lines"},
	})

	if len(info.EmbeddedAssets) > 0 {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "embedded assets (not copied)",
			Chose:   "skip COPY for " + strings.Join(info.EmbeddedAssets, ", "),
			Because: "these are baked into the binary via //go:embed, so copying them again would only bloat the image",
		})
	}
}

func chooseUser(plan *types.BuildPlan) {
	plan.RuntimeStage.User = "nonroot"
	plan.Decisions = append(plan.Decisions, types.Decision{
		Topic:        "runtime user",
		Chose:        "nonroot (UID 65532)",
		Because:      "running as non-root is a security best practice and is built into the distroless :nonroot images",
		Alternatives: []string{"root (insecure)"},
	})
}

func choosePorts(info *types.ProjectInfo, plan *types.BuildPlan) {
	ports := info.ExposedPorts
	if len(ports) == 0 {
		ports = []int{8080}
		reason := "no listen port found in source"
		if info.PortFromEnv {
			reason = "the port is read from a PORT environment variable; 8080 is the documented default"
		}
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "exposed port",
			Chose:   "8080 (default)",
			Because: reason,
		})
	} else {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "exposed port",
			Chose:   joinInts(ports),
			Because: "inferred from listen addresses found in the source",
		})
	}
	plan.RuntimeStage.ExposedPorts = ports

	if info.PortFromEnv {
		plan.RuntimeStage.Env = append(plan.RuntimeStage.Env, types.EnvVar{
			Key:   "PORT",
			Value: strconv.Itoa(ports[0]),
		})
	}
}

func chooseFrameworkEnv(info *types.ProjectInfo, plan *types.BuildPlan) {
	switch info.Framework {
	case "gin":
		plan.RuntimeStage.Env = append(plan.RuntimeStage.Env, types.EnvVar{Key: "GIN_MODE", Value: "release"})
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "framework configuration",
			Chose:   "GIN_MODE=release",
			Because: "the Gin framework was detected; release mode disables debug logging and the startup warning in production",
		})
	case "fiber", "echo", "chi", "gorilla", "fasthttp", "beego", "iris":
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "framework detected",
			Chose:   info.Framework,
			Because: "recognized the web framework from imports; no special runtime env is required for it",
		})
	}
}

func noteRuntimeNeeds(info *types.ProjectInfo, plan *types.BuildPlan) {
	if info.NeedsCACerts {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "CA certificates",
			Chose:   "provided by the distroless base (no extra step)",
			Because: "the program makes outbound TLS calls (HTTP client / DB driver / crypto/tls); the chosen distroless image already bundles the CA bundle, so it will not fail with x509 errors",
		})
	}
	if info.NeedsTZData {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "timezone database",
			Chose:   "provided by the distroless base (no extra step)",
			Because: "the program calls time.LoadLocation; the distroless image bundles tzdata, so timezone lookups will succeed",
		})
	}
}

func noteWorkspace(info *types.ProjectInfo, plan *types.BuildPlan) {
	if !info.IsWorkspace {
		return
	}
	plan.Decisions = append(plan.Decisions, types.Decision{
		Topic:   "go workspace",
		Chose:   "copy go.work and build the whole workspace",
		Because: fmt.Sprintf("a go.work file spanning %d module(s) drives this build; the Dockerfile copies it so `go build` resolves every module", len(info.WorkspaceModules)),
	})
}

func noteBuildTags(info *types.ProjectInfo, plan *types.BuildPlan) {
	if len(info.BuildTags) == 0 {
		return
	}
	plan.Decisions = append(plan.Decisions, types.Decision{
		Topic:        "build tags",
		Chose:        "not passed automatically",
		Because:      "custom build constraints were found (" + strings.Join(info.BuildTags, ", ") + "); whether they should be active is project-specific, so lbd leaves them off. Add `-tags=<name>` to the build command if a tag is required.",
		Alternatives: []string{"pass -tags=" + strings.Join(info.BuildTags, ",")},
	})
}

func noteCacheMounts(plan *types.BuildPlan) {
	if !plan.UseCacheMounts {
		return
	}
	plan.Decisions = append(plan.Decisions, types.Decision{
		Topic:   "build cache",
		Chose:   "BuildKit cache mounts for /go/pkg/mod and /root/.cache/go-build",
		Because: "reusing the module and compile caches across builds makes incremental rebuilds dramatically faster without enlarging the final image",
	})
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
	plan.BuildStage.SystemPackages = dedupe(buildPkgs)

	if len(plan.BuildStage.SystemPackages) > 0 {
		plan.Decisions = append(plan.Decisions, types.Decision{
			Topic:   "build-time system packages",
			Chose:   strings.Join(plan.BuildStage.SystemPackages, ", "),
			Because: "required by Go dependencies that link against C libraries (sourced from the leanBuildDocker knowledge base)",
		})
	}
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ", ")
}

func dedupe(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
