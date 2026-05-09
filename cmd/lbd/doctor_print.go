package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/doctor"
)

func printDoctorReport(w io.Writer, r *doctor.Report) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Doctor Report")
	fmt.Fprintln(w, strings.Repeat("-", 13))
	fmt.Fprintf(w, "Project:    %s\n", r.ProjectDir)
	if r.DockerfileExists {
		fmt.Fprintf(w, "Dockerfile: %s\n", r.DockerfilePath)
	} else {
		fmt.Fprintln(w, "Dockerfile: (none)")
	}
	fmt.Fprintln(w)

	if len(r.Findings) == 0 {
		fmt.Fprintln(w, "[OK] no issues found")
		return
	}

	for _, f := range r.Findings {
		fmt.Fprintf(w, "[%s] %s — %s\n", f.Severity, f.Topic, f.Message)
		if f.Suggest != "" {
			fmt.Fprintf(w, "       > %s\n", f.Suggest)
		}
	}
}
