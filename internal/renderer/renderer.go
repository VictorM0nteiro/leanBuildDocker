package renderer

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

//go:embed templates/*
var templatesFS embed.FS

// Options controls how Render behaves.
type Options struct {
	// Force overwrites existing output files. Without this, Render fails
	// if any target file already exists, to avoid clobbering hand-edited
	// Dockerfiles.
	Force bool
}

// outputFile pairs a destination filename with the embedded template that
// produces it. Adding a new artifact is one entry here plus one .tmpl file.
type outputFile struct {
	name     string
	template string
}

var goOutputs = []outputFile{
	{"Dockerfile", "templates/golang/dockerfile.tmpl"},
	{".dockerignore", "templates/golang/dockerignore.tmpl"},
	{".lbd-report.md", "templates/report.tmpl"},
}

// Render writes the generated artifacts to outputDir.
// All templates are rendered in memory first; nothing is written if any
// render fails. Existing files are preserved unless opts.Force is true.
func Render(plan *types.BuildPlan, outputDir string, opts Options) error {
	rendered := make(map[string]string, len(goOutputs))

	for _, f := range goOutputs {
		content, err := renderOne(f.template, plan)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", f.name, err)
		}
		rendered[f.name] = content
	}

	if !opts.Force {
		for _, f := range goOutputs {
			path := filepath.Join(outputDir, f.name)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists; pass --force to overwrite", f.name)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("checking %s: %w", path, err)
			}
		}
	}

	for name, content := range rendered {
		if err := writeFile(outputDir, name, content); err != nil {
			return err
		}
	}
	return nil
}

func renderOne(path string, plan *types.BuildPlan) (string, error) {
	funcs := template.FuncMap{
    "execForm": execForm,
    "join":     strings.Join,
}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcs).ParseFS(templatesFS, path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, plan); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func execForm(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = fmt.Sprintf("%q", a)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func writeFile(dir, name, content string) error {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
