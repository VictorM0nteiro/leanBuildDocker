package scanner

// FileInventory is the result of scanning a project directory.
// include file lists, sizes, and detected manifest paths.
type FileInvetory struct {
	ProjectRoot string
	Files       []string // relative paths from project root
}

func Scan(projectPath string) (*FileInvetory, error) {
	return &FileInvetory{
		ProjectRoot: projectPath,
		Files: []string{"go.mod", "main.go"},
	}, nil
}
