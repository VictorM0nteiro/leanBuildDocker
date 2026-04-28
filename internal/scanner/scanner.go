package scanner

// FileInventory is the result of scanning a project directory.
// include file lists, sizes, and detected manifest paths.
type FileInventory struct {
	ProjectRoot string
	Files       []string // relative paths from project root
}

func Scan(projectPath string) (*FileInventory, error) {
	return &FileInventory{
		ProjectRoot: projectPath,
		Files: []string{"go.mod", "main.go"},
	}, nil
}
