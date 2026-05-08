package validator

import "strings"

// Finding is a heuristic-driven hint about a smoke-test failure.
type Finding struct {
	Severity string // "warn" | "error"
	Message  string
	Suggest  string
}

// AnalyzeStderr returns hints based on common container-startup failures.
// The patterns are conservative: only hits we're confident about produce
// suggestions, to avoid noisy false positives.
func AnalyzeStderr(stderr string) []Finding {
	var fs []Finding

	if strings.Contains(stderr, "no such file or directory") {
		fs = append(fs, Finding{
			Severity: "error",
			Message:  "container reported 'no such file or directory'",
			Suggest:  "verify the binary was copied correctly and that the runtime base image has the required loader (use distroless/base for CGO, distroless/static for static binaries)",
		})
	}
	if strings.Contains(stderr, "cannot open shared object file") {
		fs = append(fs, Finding{
			Severity: "error",
			Message:  "container can't load a shared library",
			Suggest:  "binary was linked dynamically but runtime image lacks the loader; switch to distroless/base (glibc) or build with CGO_ENABLED=0",
		})
	}
	if strings.Contains(stderr, "exec format error") {
		fs = append(fs, Finding{
			Severity: "error",
			Message:  "exec format error",
			Suggest:  "architecture mismatch — binary was built for a different CPU architecture than the container",
		})
	}
	return fs
}
