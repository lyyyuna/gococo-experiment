package compile

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lyyyuna/gococo/pkg/log"
	"github.com/lyyyuna/gococo/pkg/util"
	"golang.org/x/exp/maps"
	"golang.org/x/mod/modfile"
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

func (c *Compile) readGoMod() string {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		log.Fatalf("fail to read GOMOD: %v", err)
	}

	return strings.TrimSpace(string(out))
}

func (c *Compile) readProjectMetaInfo() {

	c.envGOWORK = c.readGoWork()
	c.envGOMOD = c.readGoMod()

	if c.envGOMOD == "/dev/null" && c.envGOWORK == "" {
		log.Fatalf("gococo only support go mod project")
	}

	if c.envGOWORK != "" {
		c.isGoWork = true
	}

	// find project root dir
	if c.isGoWork {
		c.curProjectRootDir = filepath.Dir(c.envGOWORK)
	} else {
		c.curProjectRootDir = filepath.Dir(c.envGOMOD)
	}

	if !util.SubElem(c.curProjectRootDir, c.curWd) {
		log.Fatalf("you should execute gococo in the project directory")
	}

	// find all importpaths in the project
	modulePaths := make(map[string]*modProject)
	if c.isGoWork {
		modulePaths = c.getModulePathsFromGoWork(c.envGOWORK)
	} else {
		p := c.getModulePathFromGoMod(c.envGOMOD)
		modulePaths[p.modulePath] = p
	}
	c.modulePaths = modulePaths

	c.pkgs = make(map[string]*Package)
	for _, m := range modulePaths {
		maps.Copy(c.pkgs, c.listPackages(m.path))
	}
}

func (c *Compile) displayProjectMetaInfo() {
	log.Infof("project root directory: %v", c.curProjectRootDir)

	if c.envGOWORK != "" {
		log.Infof("go workspace is enabled")
	}

	for k, v := range c.modulePaths {
		if v.isVendor {
			log.Infof("the [%v] is built with vendor enabled", k)
		}
	}
}

// transformMetaInfoInCache, transforms the original meta info to match the temporary directory
func (c *Compile) transformMetaInfoInCache(tmpBase string) {
	c.tmpProjectRootDir = tmpBase
	// we have checked before, so ignore err here
	rel, _ := filepath.Rel(c.curProjectRootDir, c.curWd)
	c.tmpWd = filepath.Join(c.tmpProjectRootDir, rel)

	for _, m := range c.modulePaths {
		if m.inRootProject {
			// we have checked before, so ignore err here
			rel, _ := filepath.Rel(c.curProjectRootDir, m.path)
			m.tmpPath = filepath.Join(c.curProjectRootDir, rel)

			rel, _ = filepath.Rel(c.curProjectRootDir, m.gomodPath)
			m.tmpGomodPath = filepath.Join(c.curProjectRootDir, rel)
		}
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
		log.Fatalf("execute go list failed, err: %v, stdout: \n%v\n, stderr: \n%v\n", err, string(out), errBuf.String())
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
func (c *Compile) checkIfVendor(path string) bool {
	if c.buildMod == "vendor" {
		return true
	}

	vendorDir := filepath.Join(path, "vendor")
	if _, err := os.Stat(vendorDir); err != nil {
		return false
	} else {
		return true
	}
}

// getModulePathFromGoMod get module path from the go.mod file
func (c *Compile) getModulePathFromGoMod(path string) *modProject {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("cannot read the go.mod file: %v", err)
	}
	goModFile, err := modfile.Parse(path, buf, nil)
	if err != nil {
		log.Fatalf("cannot parse go.mod: %v", err)
	}

	absPath := filepath.Dir(path)
	var inRootProject bool
	if util.SubElem(absPath, c.curProjectRootDir) {
		inRootProject = true
	}
	return &modProject{
		path:          absPath,
		gomodPath:     path,
		inRootProject: inRootProject,
		modulePath:    goModFile.Module.Mod.Path,
		isVendor:      c.checkIfVendor(path),
	}
}

// getModulePathFromGoWork get module paths from the go.work file
func (c *Compile) getModulePathsFromGoWork(path string) map[string]*modProject {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("cannot read the go.work file: %v", err)
	}

	goWorkFile, err := modfile.ParseWork(path, buf, nil)
	if err != nil {
		log.Fatalf("cannot parse go.work: %v", err)
	}

	modulePaths := make(map[string]*modProject)
	for _, use := range goWorkFile.Use {
		var subModAbsDir string

		if filepath.IsAbs(subModAbsDir) {

		} else {
			subModAbsDir = filepath.Join(c.curProjectRootDir, use.Path)
		}

		if !util.SubElem(c.curProjectRootDir, subModAbsDir) {
			log.Fatalf("sub mod project not in the same directory")
		}
		subModFilePath := filepath.Join(use.Path, "go.mod")

		modproject := c.getModulePathFromGoMod(subModFilePath)
		modulePaths[modproject.modulePath] = modproject
	}

	return modulePaths
}

// updateGoModGoWorkFile rewrites the go.mod/go.work file in the temporary directory,
//
// if it has a 'replace' directive, and the directive has a relative local path,
// it will be rewritten with a absolute path.
//
// ex.
//
// suppose original project is located at /path/to/aa/bb/cc, go.mod contains a directive:
// 'replace github.com/qiniu/bar => ../home/foo/bar'
//
// after the project is copied to temporary directory, it should be rewritten as
// 'replace github.com/qiniu/bar => /path/to/aa/bb/home/foo/bar'
func (c *Compile) updateGoModGoWorkFile() {

	// rewrite go.work
	if c.isGoWork {
		tempWorkfile := filepath.Join(c.tmpProjectRootDir, "go.work")
		buf, err := ioutil.ReadFile(tempWorkfile)
		if err != nil {
			log.Fatalf("cannot find go.work file in temporary directory: %v", err)
		}
		oriGoWorkFile, err := modfile.ParseWork(tempWorkfile, buf, nil)
		if err != nil {
			log.Fatalf("cannot parse go.work: %v", err)
		}

		updateFlag := false
		for index := range oriGoWorkFile.Replace {
			replace := oriGoWorkFile.Replace[index]
			oldPath := replace.Old.Path
			oldVersion := replace.Old.Version
			oriNewPath := replace.New.Path
			newVersion := replace.New.Version

			// has version, means it replace to a network dependency
			if newVersion != "" {
				continue
			}
			// no version, means it replace to a local filesystem

			var newPath string
			// tansform all to abs path
			if !filepath.IsAbs(oriNewPath) {
				newPath, _ = filepath.Abs(filepath.Join(c.curProjectRootDir, oriNewPath))
			}
			// rewrite path which replace to self dir
			if util.SubElem(c.curProjectRootDir, newPath) {
				rel, _ := filepath.Rel(c.curProjectRootDir, newPath)
				newPath = filepath.Join(c.tmpProjectRootDir, rel)
			}

			// DropReplace & AddReplace will not return error
			// so no need to check the error
			_ = oriGoWorkFile.DropReplace(oldPath, oldVersion)
			_ = oriGoWorkFile.AddReplace(oldPath, oldVersion, newPath, newVersion)
			updateFlag = true
		}

		oriGoWorkFile.Cleanup()
		// Format will not return error, so ignore the returned error
		// func (f *File) Format() ([]byte, error) {
		//     return Format(f.Syntax), nil
		// }
		newWorkFile := modfile.Format(oriGoWorkFile.Syntax)

		if updateFlag {
			log.Infof("[%v] needs rewrite", tempWorkfile)
			err := os.WriteFile(tempWorkfile, newWorkFile, os.ModePerm)
			if err != nil {
				log.Fatalf("fail to update go.work: %v", err)
			}
		}
	}

	// rewrite all go.mod
	for _, m := range c.modulePaths {

		if !m.inRootProject {
			continue
		} else {
			// 1. rewrite dependency relative path's to absolute path
		}

		tempModfile := m.tmpGomodPath
		buf, err := ioutil.ReadFile(tempModfile)
		if err != nil {
			log.Fatalf("cannot find go.mod file in temporary directory: %v", err)
		}
		oriGoModFile, err := modfile.Parse(tempModfile, buf, nil)
		if err != nil {
			log.Fatalf("cannot parse go.mod: %v", err)
		}

		updateFlag := false
		for index := range oriGoModFile.Replace {
			replace := oriGoModFile.Replace[index]
			oldPath := replace.Old.Path
			oldVersion := replace.Old.Version
			oriNewPath := replace.New.Path
			newVersion := replace.New.Version

			// has version, means it replace to a network dependency
			if newVersion != "" {
				continue
			}
			// no version, means it replace to a local filesystem

			var newPath string
			// tansform all to abs path
			if !filepath.IsAbs(oriNewPath) {
				newPath, _ = filepath.Abs(filepath.Join(m.path, oriNewPath))
			}
			// rewrite path which replace to self dir
			if util.SubElem(m.path, newPath) {
				rel, _ := filepath.Rel(m.path, newPath)
				newPath = filepath.Join(m.tmpPath, rel)
			}

			// DropReplace & AddReplace will not return error
			// so no need to check the error
			_ = oriGoModFile.DropReplace(oldPath, oldVersion)
			_ = oriGoModFile.AddReplace(oldPath, oldVersion, newPath, newVersion)
			updateFlag = true
		}
		oriGoModFile.Cleanup()
		// Format will not return error, so ignore the returned error
		// func (f *File) Format() ([]byte, error) {
		//     return Format(f.Syntax), nil
		// }
		newModFile, _ := oriGoModFile.Format()

		if updateFlag {
			log.Infof("[%v] needs rewrite", m.gomodPath)
			err := os.WriteFile(tempModfile, newModFile, os.ModePerm)
			if err != nil {
				log.Fatalf("fail to update go.mod: %v", err)
			}
		}
	}
}
