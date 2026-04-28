package types

// ProjectInfo represents facts discovered about a project by an Analyzer.
type ProjectInfo struct {
	Language        string
	LanguageVersion string
	Dependencies    []string
	EntryPoint      string
	// HasCGO indicates whether the project uses cgo. This single boolean
	// drives major Planner decisions: with cgo, the runtime image cannot
	// be "scratch" or distroless/static; it needs glibc or musl available.
	HasCGO          bool
}
