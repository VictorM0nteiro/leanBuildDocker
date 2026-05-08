package types

// ProjectInfo represents facts discovered about a project by an Analyzer.
type ProjectInfo struct {
	Language        string
	LanguageVersion string
	ModulePath      string
	ProjectRoot     string

	DirectDependencies []GoDep

	HasCGO        bool
	BuildTags     []string
	UsesVendoring bool

	MainPackages []string
	EntryPoint   string

	Warnings []Warning
}

// GoDep is a dependency entry from go.mod.
type GoDep struct {
	Path     string
	Version  string
	Indirect bool
}

// Warning is a non-fatal issue discovered during analysis.
type Warning struct {
	Message string
}
