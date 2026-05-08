package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/validator"
)

func printValidationReport(w io.Writer, r *validator.Result) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Validation Report")
	fmt.Fprintln(w, strings.Repeat("-", 17))

	fmt.Fprintf(w, "%s Build\n", check(r.Build.OK))
	if !r.Build.OK && r.Build.Err != nil {
		fmt.Fprintf(w, "   %v\n", r.Build.Err)
	}

	if r.ImageSizeBytes > 0 {
		fmt.Fprintf(w, "  Image size: %s\n", formatSize(r.ImageSizeBytes))
	}

	if r.Build.OK {
		fmt.Fprintf(w, "%s Smoke test\n", check(r.Smoke.OK))
		if !r.Smoke.OK {
			if r.Smoke.Logs != "" {
				fmt.Fprintf(w, "   logs: %s\n", trimLines(r.Smoke.Logs, 3))
			}
			for _, f := range r.Smoke.Findings {
				fmt.Fprintf(w, "   > %s\n", f.Suggest)
			}
		}
	}

	switch {
	case !r.HadolintRan:
		fmt.Fprintln(w, "  Hadolint: not installed (skipped)")
	case len(r.HadolintIssues) == 0:
		fmt.Fprintln(w, "✓ Hadolint")
	default:
		fmt.Fprintf(w, "%s Hadolint (%d issue(s))\n", check(false), len(r.HadolintIssues))
		for _, i := range r.HadolintIssues {
			fmt.Fprintf(w, "   • %s\n", i)
		}
	}

	fmt.Fprintln(w, "")
	if r.Cleaned {
		fmt.Fprintf(w, "Image %s removed.\n", r.ImageTag)
	} else if r.ImageTag != "" {
		fmt.Fprintf(w, "Image kept: %s\n", r.ImageTag)
	}
}

func check(ok bool) string {
	if ok {
		return "[OK]"
	}
	return "[X] "
}

func formatSize(b int64) string {
	const mb = 1024 * 1024
	if b < mb {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/mb)
}

func trimLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= n {
		return strings.Join(lines, " | ")
	}
	return strings.Join(lines[:n], " | ") + " ..."
}
