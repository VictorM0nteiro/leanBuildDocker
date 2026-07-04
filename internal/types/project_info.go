package types

// ProjectInfo represents facts discovered about a project by an Analyzer.
// It is the output of any Analyzer and the sole input to the Planner —
// no Go-specific types leak past this boundary except the language
// extension fields below, which the Planner reads through explicit checks.
type ProjectInfo struct {
	Language        string
	LanguageVersion string
	ModulePath      string
	ProjectRoot     string

	DirectDependencies []GoDep

	HasCGO        bool
	BuildTags     []string
	UsesVendoring bool

	// IsWorkspace is true when a go.work file drives a multi-module build.
	IsWorkspace      bool
	WorkspaceModules []string

	MainPackages []string
	EntryPoint   string

	// Framework is the detected web framework (gin, echo, chi, fiber,
	// gorilla, fasthttp, net/http) or "" when the project is not a server.
	Framework string

	// ExposedPorts are TCP ports the program listens on, inferred from
	// source (string literals like ":8080", framework .Run calls, or a
	// PORT environment lookup). Empty when nothing could be inferred.
	ExposedPorts []int

	// PortFromEnv is true when the listen port is read from an environment
	// variable (e.g. os.Getenv("PORT")) rather than a fixed literal.
	PortFromEnv bool

	// RuntimeAssets are directories/files the program reads from disk at
	// runtime (templates, static, migrations, config, ...). They must be
	// copied into the runtime image. Assets covered by //go:embed are
	// excluded here and reported in EmbeddedAssets instead.
	RuntimeAssets  []RuntimeAsset
	EmbeddedAssets []string

	// NeedsCACerts is true when the program makes outbound TLS calls
	// (HTTP client, database driver, crypto/tls) and therefore needs a
	// CA bundle in the runtime image.
	NeedsCACerts bool

	// NeedsTZData is true when the program loads timezone data at runtime
	// (time.LoadLocation) and therefore needs the zoneinfo database.
	NeedsTZData bool

	// VersionVar, when non-empty, is a linker path (e.g. "main.version")
	// of a package-level string variable the build can stamp via -ldflags -X.
	VersionVar string

	Warnings           []Warning
	SystemPackageHints []SystemPackageHint
}

// GoDep is a dependency entry from go.mod.
type GoDep struct {
	Path     string
	Version  string
	Indirect bool
}

// RuntimeAsset is a path (relative to the project root) that the program
// reads at runtime and that must therefore be copied into the runtime
// image. Kind is a short human label used in the report ("templates",
// "migrations", "config", ...).
type RuntimeAsset struct {
	Path string
	Kind string
	IsDir bool
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
