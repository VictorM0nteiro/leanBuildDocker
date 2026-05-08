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
	SystemPackageHints []SystemPackageHint

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

// SystemPackageHint is a knowledge-base lookup result for a single dep.
// The Planner translates these into concrete install commands based on
// the chosen base image (Alpine vs Debian).
type SystemPackageHint struct {
	DepPath string
	Build   PackageSet
	Runtime PackageSet
}