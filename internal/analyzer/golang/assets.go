package golang

import (
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
	"github.com/VictorM0nteiro/leanBuildDocker/internal/types"
)

// assetDirKinds maps a conventional top-level directory name to the human
// "kind" label used in the report. These are directories a Go program
// typically reads from disk at runtime (HTML templates, static files, SQL
// migrations, config).
var assetDirKinds = map[string]string{
	"templates":    "html templates",
	"template":     "html templates",
	"views":        "html templates",
	"static":       "static assets",
	"public":       "static assets",
	"assets":       "static assets",
	"dist":         "static assets",
	"web":          "web assets",
	"migrations":   "database migrations",
	"migration":    "database migrations",
	"config":       "config files",
	"configs":      "config files",
	"locales":      "i18n bundles",
	"i18n":         "i18n bundles",
	"translations": "i18n bundles",
}

// assetFileKinds maps conventional top-level config filenames to a kind.
var assetFileKinds = map[string]string{
	"config.yaml": "config file",
	"config.yml":  "config file",
	"config.json": "config file",
	"config.toml": "config file",
	"app.yaml":    "config file",
	"app.yml":     "config file",
}

// detectAssets decides which project paths must be copied into the runtime
// image. The core judgement — and the reason this is real analysis rather
// than a guess — is the //go:embed check: an asset directory that the source
// embeds is already inside the binary, so copying it would be redundant. It
// goes to EmbeddedAssets (informational) instead of RuntimeAssets.
//
// Only top-level entries are considered. Runtime assets in idiomatic Go live
// beside the module root (or the entry point), not buried inside internal/.
func detectAssets(inv *scanner.FileInventory, facts *sourceFacts) (assets []types.RuntimeAsset, embedded []string, warnings []types.Warning) {
	for _, dir := range inv.Dirs {
		if strings.Contains(dir, "/") {
			continue // top-level only
		}
		kind, ok := assetDirKinds[dir]
		if !ok {
			continue
		}
		if facts.embedRoots[dir] {
			embedded = append(embedded, dir+"/")
			continue
		}
		assets = append(assets, types.RuntimeAsset{Path: dir, Kind: kind, IsDir: true})
	}

	for _, file := range inv.Files {
		if strings.Contains(file, "/") {
			continue // top-level only
		}
		// .env is intentionally never copied: it usually holds secrets and is
		// gitignored. Surface it as a warning so the user mounts it instead.
		if file == ".env" || strings.HasPrefix(file, ".env.") {
			warnings = append(warnings, types.Warning{
				Message: "found " + file + " at project root — not copied into the image (may contain secrets). Mount it at runtime or use environment variables.",
			})
			continue
		}
		kind, ok := assetFileKinds[file]
		if !ok {
			continue
		}
		if facts.embedRoots[file] {
			embedded = append(embedded, file)
			continue
		}
		assets = append(assets, types.RuntimeAsset{Path: file, Kind: kind, IsDir: false})
	}

	return assets, embedded, warnings
}
