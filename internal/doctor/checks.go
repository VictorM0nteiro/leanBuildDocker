package doctor

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

var (
	fromRe = regexp.MustCompile(`(?mi)^\s*FROM\s+(\S+)`)
	userRe = regexp.MustCompile(`(?mi)^\s*USER\s+(.+)$`)
)

// lastFROM returns the last FROM image (i.e., the runtime stage in a multi-stage build).
func lastFROM(dockerfile string) string {
	m := fromRe.FindAllStringSubmatch(dockerfile, -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

func lastUSER(dockerfile string) string {
	m := userRe.FindAllStringSubmatch(dockerfile, -1)
	if len(m) == 0 {
		return ""
	}
	return strings.TrimSpace(m[len(m)-1][1])
}

func checkCGOMismatch(info *types.ProjectInfo, dockerfile string) []Finding {
	if !info.HasCGO {
		return nil
	}
	runtime := lastFROM(dockerfile)
	if strings.Contains(runtime, "scratch") || strings.Contains(runtime, "distroless/static") {
		return []Finding{{
			Severity: SeverityError,
			Topic:    "CGO + static runtime",
			Message:  "project uses CGO but runtime base is " + runtime,
			Suggest:  "switch to distroless/base (glibc) or build with CGO_ENABLED=0",
		}}
	}
	return nil
}

func checkRuntimeUser(dockerfile string) []Finding {
	user := lastUSER(dockerfile)
	if user == "" {
		return []Finding{{
			Severity: SeverityWarn,
			Topic:    "runtime user",
			Message:  "no USER directive — container will run as root",
			Suggest:  "add `USER nonroot` (distroless) or create a non-root user explicitly",
		}}
	}
	if user == "root" || user == "0" || user == "0:0" {
		return []Finding{{
			Severity: SeverityWarn,
			Topic:    "runtime user",
			Message:  "container runs as root",
			Suggest:  "switch to a non-root user for better security",
		}}
	}
	return nil
}

func checkCACerts(projectDir, dockerfile string) []Finding {
	if !projectImportsNetHTTP(projectDir) {
		return nil
	}
	runtime := lastFROM(dockerfile)
	if strings.Contains(runtime, "distroless") {
		return nil // distroless ships ca-certificates
	}
	if strings.Contains(runtime, "scratch") {
		return []Finding{{
			Severity: SeverityWarn,
			Topic:    "CA certificates",
			Message:  "project uses net/http but runtime is scratch (no CA certs)",
			Suggest:  "COPY /etc/ssl/certs from builder, or switch to distroless/static",
		}}
	}
	if strings.Contains(runtime, "alpine") && !strings.Contains(dockerfile, "ca-certificates") {
		return []Finding{{
			Severity: SeverityWarn,
			Topic:    "CA certificates",
			Message:  "alpine runtime without ca-certificates — outbound HTTPS will fail",
			Suggest:  "add `RUN apk add --no-cache ca-certificates`",
		}}
	}
	return nil
}

func checkExpose(dockerfile string) []Finding {
	if strings.Contains(strings.ToUpper(dockerfile), "\nEXPOSE ") || strings.HasPrefix(strings.ToUpper(dockerfile), "EXPOSE ") {
		return nil
	}
	return []Finding{{
		Severity: SeverityInfo,
		Topic:    "exposed port",
		Message:  "no EXPOSE directive in Dockerfile",
		Suggest:  "add `EXPOSE <port>` for documentation and tooling (e.g., `docker ps`)",
	}}
}

func checkDockerignore(projectDir string) []Finding {
	hasTests := walkHasTests(projectDir)
	if !hasTests {
		return nil
	}

	diBytes, err := os.ReadFile(filepath.Join(projectDir, ".dockerignore"))
	if err != nil {
		return []Finding{{
			Severity: SeverityWarn,
			Topic:    ".dockerignore",
			Message:  "project has *_test.go files but no .dockerignore",
			Suggest:  "create one excluding tests, fixtures, and editor metadata",
		}}
	}
	di := string(diBytes)
	if !strings.Contains(di, "_test.go") {
		return []Finding{{
			Severity: SeverityInfo,
			Topic:    ".dockerignore",
			Message:  ".dockerignore exists but doesn't exclude *_test.go",
			Suggest:  "add `*_test.go` to keep test files out of the build context",
		}}
	}
	return nil
}

// walkHasTests returns true if any *_test.go file exists in the project,
// excluding vendor/, testdata/, and dotted directories.
func walkHasTests(projectDir string) bool {
	var found bool
	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// projectImportsNetHTTP scans .go files (skipping tests and vendor) for
// an import of "net/http" using the AST — no regex on source.
func projectImportsNetHTTP(projectDir string) bool {
	var found bool
	fset := token.NewFileSet()
	_ = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return nil
		}
		for _, imp := range f.Imports {
			if imp.Path.Value == `"net/http"` {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}
