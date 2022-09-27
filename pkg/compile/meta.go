package compile

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lyyyuna/gococo/pkg/log"
)

// PackageCover holds all the generate coverage variables of a package
type PackageCover struct {
	Package *Package
	Vars    map[string]*FileVar
}

// FileVar holds the name of the generated coverage variables targeting the named file.
type FileVar struct {
	File string // importpath + filename
	Var  string
}

// Package map a package output by go list
// this is subset of package struct in: https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go#L58
type Package struct {
	Dir        string `json:"Dir"`        // directory containing package sources
	ImportPath string `json:"ImportPath"` // import path of package in dir
	Name       string `json:"Name"`       // package name
	Target     string `json:",omitempty"` // installed target for this package (may be executable)
	Root       string `json:",omitempty"` // Go root, Go path dir, or module root dir containing this package

	Module   *ModulePublic `json:",omitempty"`         // info about package's module, if any
	Goroot   bool          `json:"Goroot,omitempty"`   // is this package in the Go root?
	Standard bool          `json:"Standard,omitempty"` // is this package part of the standard Go library?
	DepOnly  bool          `json:"DepOnly,omitempty"`  // package is only a dependency, not explicitly listed

	// Source files
	// If you add to this list you MUST add to p.AllFiles (below) too.
	// Otherwise file name security lists will not apply to any new additions.
	GoFiles           []string `json:",omitempty"` // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles          []string `json:",omitempty"` // .go source files that import "C"
	CompiledGoFiles   []string `json:",omitempty"` // .go output from running cgo on CgoFiles
	IgnoredGoFiles    []string `json:",omitempty"` // .go source files ignored due to build constraints
	InvalidGoFiles    []string `json:",omitempty"` // .go source files with detected problems (parse error, wrong package name, and so on)
	IgnoredOtherFiles []string `json:",omitempty"` // non-.go source files ignored due to build constraints
	CFiles            []string `json:",omitempty"` // .c source files
	CXXFiles          []string `json:",omitempty"` // .cc, .cpp and .cxx source files
	MFiles            []string `json:",omitempty"` // .m source files
	HFiles            []string `json:",omitempty"` // .h, .hh, .hpp and .hxx source files
	FFiles            []string `json:",omitempty"` // .f, .F, .for and .f90 Fortran source files
	SFiles            []string `json:",omitempty"` // .s source files
	SwigFiles         []string `json:",omitempty"` // .swig files
	SwigCXXFiles      []string `json:",omitempty"` // .swigcxx files
	SysoFiles         []string `json:",omitempty"` // .syso system object files added to package

	// Embedded files
	EmbedPatterns []string `json:",omitempty"` // //go:embed patterns
	EmbedFiles    []string `json:",omitempty"` // files matched by EmbedPatterns

	// Dependency information
	Deps      []string          `json:"Deps,omitempty"` // all (recursively) imported dependencies
	Imports   []string          `json:",omitempty"`     // import paths used by this package
	ImportMap map[string]string `json:",omitempty"`     // map from source import to ImportPath (identity entries omitted)

	// Error information
	Incomplete bool            `json:"Incomplete,omitempty"` // this package or a dependency has an error
	Error      *PackageError   `json:"Error,omitempty"`      // error loading package
	DepsErrors []*PackageError `json:"DepsErrors,omitempty"` // errors loading dependencies
}

// ModulePublic represents the package info of a module
type ModulePublic struct {
	Path      string        `json:",omitempty"` // module path
	Version   string        `json:",omitempty"` // module version
	Versions  []string      `json:",omitempty"` // available module versions
	Replace   *ModulePublic `json:",omitempty"` // replaced by this module
	Time      *time.Time    `json:",omitempty"` // time version was created
	Update    *ModulePublic `json:",omitempty"` // available update (with -u)
	Main      bool          `json:",omitempty"` // is this the main module?
	Indirect  bool          `json:",omitempty"` // module is only indirectly needed by main module
	Dir       string        `json:",omitempty"` // directory holding local copy of files, if any
	GoMod     string        `json:",omitempty"` // path to go.mod file describing module, if any
	GoVersion string        `json:",omitempty"` // go version used in module
	Error     *ModuleError  `json:",omitempty"` // error loading module
}

// ModuleError represents the error loading module
type ModuleError struct {
	Err string // error text
}

// PackageError is the error info for a package when list failed
type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}

func (c *Compile) readGoWork() string {
	out, err := exec.Command("go", "env", "GOWORK").Output()
	if err != nil {
		log.Fatalf("fail to read GOWORK: %v", err)
	}

	return strings.TrimSpace(string(out))
}

func (c *Compile) readProjectMetaInfo() {
	c.curGoWork = c.readGoWork()

	pkgs := c.listPackages(c.curWd)
	for _, pkg := range pkgs {
		// check if go mod is enabled
		if pkg.Module == nil {
			log.Fatalf("gococo only support go mod project")
		}

		c.curProjectRootDir = pkg.Module.Dir
		c.projectModulePath = pkg.Module.Path

		// no need to loop each package
		break
	}

	// need package info for the whole project, not only the current working directory
	if c.curWd != c.curProjectRootDir {
		c.pkgs = c.listPackages(c.curProjectRootDir)
	} else {
		c.pkgs = pkgs
	}

	c.isBuildModVendor = c.checkIfVendor()
	log.Donef("project meta information parsed")
}

func (c *Compile) displayProjectMetaInfo() {
	log.Infof("project root directory: %v", c.curProjectRootDir)

	if c.isBuildModVendor {
		log.Infof("build with vendor")
	}

	if c.curGoWork != "" {
		log.Infof("go workspace is enabled")
	}
}

// listPacakges uses `go list -json` command to get prjects meta information
func (c *Compile) listPackages(dir string) map[string]*Package {
	listArgs := []string{"list", "-json"}
	if c.buildTags != "" {
		listArgs = append(listArgs, "-tags", c.buildTags)
	}
	listArgs = append(listArgs, "./...")

	cmd := exec.Command("go", listArgs...)
	cmd.Dir = dir

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("execute go list -json ./... failed, err: %v, stdout: %v, stderr: %v", err, string(out), errBuf.String())
	}

	dec := json.NewDecoder(bytes.NewBuffer(out))
	pkgs := make(map[string]*Package)

	for {
		var pkg Package
		if err := dec.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("reading go list output error: %v", err)
		}
		if pkg.Error != nil {
			log.Fatalf("list package %v failed with error: %v", pkg.ImportPath, pkg.Error)
		}

		pkgs[pkg.ImportPath] = &pkg
	}

	return pkgs
}

// checkIfVendor, check go mod type based on command line or vendor directory
func (c *Compile) checkIfVendor() bool {
	if c.buildMod == "vendor" {
		return true
	}

	vendorDir := filepath.Join(c.curProjectRootDir, "vendor")
	if _, err := os.Stat(vendorDir); err != nil {
		return false
	} else {
		return true
	}
}
