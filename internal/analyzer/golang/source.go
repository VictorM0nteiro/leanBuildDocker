package golang

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/scanner"
)

// sourceFacts is everything the AST pass extracts from a project's Go source.
// One struct, one pass: every .go file is parsed a single time and its
// signals folded in here. Downstream code reads facts, never re-parses.
type sourceFacts struct {
	mainDirs    []string        // dirs (relative, forward slash) with package main + func main
	hasCGO      bool
	imports     map[string]bool // every import path seen
	frameworks  map[string]bool
	ports       map[int]bool
	portFromEnv bool
	needsCerts  bool
	needsTZ     bool
	usesEmbed   bool
	embedRoots  map[string]bool // top path segment of each //go:embed pattern
	buildTags   map[string]bool // custom build tags (GOOS/GOARCH/std filtered out)
	versionVar  string          // linker path of a version var, e.g. "main.version"
}

// frameworkByPrefix maps an import-path prefix to a framework label. Order is
// checked deterministically (see frameworkFor); the first prefix that matches
// an import wins.
var frameworkByPrefix = []struct {
	prefix string
	name   string
}{
	{"github.com/gin-gonic/gin", "gin"},
	{"github.com/labstack/echo", "echo"},
	{"github.com/go-chi/chi", "chi"},
	{"github.com/gofiber/fiber", "fiber"},
	{"github.com/gorilla/mux", "gorilla"},
	{"github.com/valyala/fasthttp", "fasthttp"},
	{"github.com/beego/beego", "beego"},
	{"github.com/kataras/iris", "iris"},
}

// outboundTLSPrefixes are import paths whose mere presence implies the program
// makes outbound TLS connections and therefore needs a CA bundle at runtime.
var outboundTLSPrefixes = []string{
	"crypto/tls",
	"database/sql",
	"github.com/aws/aws-sdk-go",
	"cloud.google.com/go",
	"google.golang.org/api",
	"google.golang.org/grpc",
	"github.com/redis/go-redis",
	"github.com/go-redis/redis",
	"go.mongodb.org/mongo-driver",
	"github.com/jackc/pgx",
	"github.com/lib/pq",
	"github.com/go-sql-driver/mysql",
	"github.com/nats-io/nats.go",
	"github.com/streadway/amqp",
	"github.com/rabbitmq/amqp091-go",
	"github.com/segmentio/kafka-go",
	"github.com/stripe/stripe-go",
	"github.com/sendgrid/sendgrid-go",
	"github.com/go-resty/resty",
	"golang.org/x/oauth2",
}

// knownEnvTags are build-constraint identifiers that come from the toolchain
// (GOOS, GOARCH, and the handful of std pseudo-tags). We drop these so
// buildTags reports only project-specific tags worth surfacing.
var knownEnvTags = map[string]bool{
	"linux": true, "darwin": true, "windows": true, "freebsd": true,
	"openbsd": true, "netbsd": true, "dragonfly": true, "solaris": true,
	"plan9": true, "aix": true, "js": true, "wasip1": true, "android": true,
	"ios": true, "unix": true,
	"amd64": true, "arm64": true, "arm": true, "386": true, "ppc64": true,
	"ppc64le": true, "mips": true, "mips64": true, "s390x": true, "riscv64": true,
	"wasm": true, "loong64": true,
	"cgo": true, "gc": true, "gccgo": true, "purego": true, "race": true,
	"ignore": true,
}

var portLiteral = regexp.MustCompile(`^[a-zA-Z0-9_.\-]*:(\d{2,5})$`)

// scanSource parses every buildable .go file in the inventory once and returns
// the folded facts. Vendored code and _test.go files are skipped: they never
// contribute to the shipped binary's entry point or runtime needs.
func scanSource(inv *scanner.FileInventory) *sourceFacts {
	f := &sourceFacts{
		imports:    map[string]bool{},
		frameworks: map[string]bool{},
		ports:      map[int]bool{},
		embedRoots: map[string]bool{},
		buildTags:  map[string]bool{},
	}
	fset := token.NewFileSet()

	for _, rel := range inv.Files {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		if isVendored(rel) || underTestdata(rel) {
			continue
		}

		src, err := os.ReadFile(inv.Abs(rel))
		if err != nil {
			continue // unreadable file: skip, don't fail the whole analysis
		}
		file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
		if err != nil {
			continue // unparsable (e.g. build-tagged for another toolchain): skip
		}

		f.foldFile(rel, file)
	}

	sort.Strings(f.mainDirs)
	return f
}

func (f *sourceFacts) foldFile(rel string, file *ast.File) {
	dir := path.Dir(rel)
	if dir == "." {
		dir = "."
	}
	isMain := file.Name.Name == "main"

	// Imports drive framework, CGO, and outbound-TLS detection.
	for _, imp := range file.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		f.imports[p] = true

		if p == "C" {
			f.hasCGO = true
		}
		if name := frameworkFor(p); name != "" {
			f.frameworks[name] = true
		}
		for _, pre := range outboundTLSPrefixes {
			if strings.HasPrefix(p, pre) {
				f.needsCerts = true
				break
			}
		}
	}

	// Comments carry //go:build constraints and //go:embed directives.
	for _, group := range file.Comments {
		for _, c := range group.List {
			f.foldComment(c.Text)
		}
	}

	// Declarations: func main (entry point) and a version var (ldflags -X).
	if isMain {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == "main" {
					f.addMainDir(dir)
				}
			case *ast.GenDecl:
				if d.Tok == token.VAR && f.versionVar == "" {
					if name := versionVarName(d); name != "" {
						f.versionVar = "main." + name
					}
				}
			}
		}
	}

	// Expression-level signals: listen ports, PORT env lookups, HTTP client
	// calls, and time.LoadLocation.
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BasicLit:
			if node.Kind == token.STRING {
				if p, ok := portFromLiteral(node.Value); ok {
					f.ports[p] = true
				}
			}
		case *ast.CallExpr:
			f.foldCall(node)
		}
		return true
	})
}

func (f *sourceFacts) foldComment(text string) {
	if strings.HasPrefix(text, "//go:embed ") {
		f.usesEmbed = true
		for _, pat := range strings.Fields(strings.TrimPrefix(text, "//go:embed ")) {
			pat = strings.Trim(pat, `"`)
			if seg := firstSegment(pat); seg != "" {
				f.embedRoots[seg] = true
			}
		}
		return
	}
	if constraint.IsGoBuild(text) {
		if expr, err := constraint.Parse(text); err == nil {
			// Eval visits every tag identifier in the expression; we use it
			// purely to enumerate tags, so the truth value is irrelevant.
			expr.Eval(func(tag string) bool {
				if !knownEnvTags[tag] && tag != "" {
					f.buildTags[tag] = true
				}
				return false
			})
		}
	}
}

func (f *sourceFacts) foldCall(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	switch pkg.Name + "." + sel.Sel.Name {
	case "os.Getenv", "os.LookupEnv":
		if isPortEnvArg(call) {
			f.portFromEnv = true
		}
	case "http.Get", "http.Post", "http.PostForm", "http.Head", "http.Do":
		f.needsCerts = true
	case "time.LoadLocation", "time.LoadLocationFromTZData":
		f.needsTZ = true
	}
}

func (f *sourceFacts) addMainDir(dir string) {
	for _, d := range f.mainDirs {
		if d == dir {
			return
		}
	}
	f.mainDirs = append(f.mainDirs, dir)
}

// frameworkFor returns the framework label for an import path, or "".
func frameworkFor(importPath string) string {
	for _, fw := range frameworkByPrefix {
		if strings.HasPrefix(importPath, fw.prefix) {
			return fw.name
		}
	}
	return ""
}

// portFromLiteral extracts a plausible listen port from a quoted string
// literal such as `":8080"`, `"0.0.0.0:9090"`, or `"localhost:3000"`.
// It deliberately rejects time-like values by requiring port >= 80.
func portFromLiteral(quoted string) (int, bool) {
	s, err := strconv.Unquote(quoted)
	if err != nil {
		return 0, false
	}
	m := portLiteral.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	p, err := strconv.Atoi(m[1])
	if err != nil || p < 80 || p > 65535 {
		return 0, false
	}
	return p, true
}

func isPortEnvArg(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	key, err := strconv.Unquote(lit.Value)
	if err != nil {
		return false
	}
	key = strings.ToUpper(key)
	return key == "PORT" || strings.HasSuffix(key, "_PORT") || strings.HasSuffix(key, "PORT")
}

func versionVarName(d *ast.GenDecl) string {
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range vs.Names {
			switch name.Name {
			case "version", "Version", "buildVersion", "BuildVersion":
				return name.Name
			}
		}
	}
	return ""
}

func firstSegment(p string) string {
	p = strings.TrimPrefix(p, "./")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func isVendored(rel string) bool {
	return rel == "vendor" || strings.HasPrefix(rel, "vendor/") || strings.Contains(rel, "/vendor/")
}

func underTestdata(rel string) bool {
	return rel == "testdata" || strings.HasPrefix(rel, "testdata/") || strings.Contains(rel, "/testdata/")
}
