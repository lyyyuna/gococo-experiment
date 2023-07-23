package compile

import (
	"crypto/sha256"
	"fmt"
	"path"
	"path/filepath"

	"github.com/lyyyuna/gococo/pkg/compile/internal/tool"
	"github.com/lyyyuna/gococo/pkg/log"
)

// inject, injects coverage counter and agent into the code
func (c *Compile) inject() {
	log.StartWait("injecting the coverage code")

	seen := make(map[string]*PackageCover)

	// the coverage counter definition in plain text
	allDeclsStr := ""

	for _, m := range c.modulePaths {
		for _, pkg := range m.pkgs {
			if pkg.Name == "main" {
				// all coverage variables' meta information of same main binary
				allCoversInMain := make([]*PackageCover, 0)
				allInjectImportPathInMain := make([]string, 0)

				// inject to main's dep
				for _, dep := range pkg.Deps {
					// injected before
					//  ex. injected in other mains' dep
					if pkgCover, ok := seen[dep]; ok {
						allCoversInMain = append(allCoversInMain, pkgCover)
						continue
					}

					// check dep pkg belongs to which module
					if pkg, ok := c.pkgs[dep]; ok {
						pkgmodulePath := pkg.Module.Path
						c.modulePaths[pkgmodulePath]
					}
				}

				// inject the main package
				mainCover, mainDecl := c.addCounters(pkg, m.injectPkgImportpath)
				// collect cover meta and cover definition
				allDeclsStr += mainDecl
				allCoversInMain = append(allCoversInMain, mainCover)
			}
		}
	}

	log.StopWait()
	log.Donef("coverage code injected")
}

// addCounters is different from official go tool cover
//
//  1. only inject covervar++ into source file
//  2. no declarartions for these covervars
//  3. return the declarations as string
func (b *Compile) addCounters(pkg *Package, gobalCoverVarImportpath string) (*PackageCover, string) {

	coverVarMap := declareCoverVars(pkg)

	decl := ""
	for file, coverVar := range coverVarMap {
		decl += "\n" + tool.Annotate(filepath.Join(pkg.tmpDir, file), coverVar.Var, gobalCoverVarImportpath) + "\n"
	}

	return &PackageCover{
		Package: pkg,
		Vars:    coverVarMap,
	}, decl
}

// declareCoverVars attaches the required cover variables names
// to the files, to be used when annotating the files.
func declareCoverVars(p *Package) map[string]*FileVar {
	coverVars := make(map[string]*FileVar)
	coverIndex := 0
	// We create the cover counters as new top-level variables in the package.
	// We need to avoid collisions with user variables (GoCover_0 is unlikely but still)
	// and more importantly with dot imports of other covered packages,
	// so we append 12 hex digits from the SHA-256 of the import path.
	// The point is only to avoid accidents, not to defeat users determined to
	// break things.
	sum := sha256.Sum256([]byte(p.ImportPath))
	h := fmt.Sprintf("%x", sum[:6])
	for _, file := range p.GoFiles {
		// These names appear in the cmd/cover HTML interface.
		var longFile = path.Join(p.ImportPath, file)
		coverVars[file] = &FileVar{
			File: longFile,
			Var:  fmt.Sprintf("GoCover_%d_%x", coverIndex, h),
		}
		coverIndex++
	}

	for _, file := range p.CgoFiles {
		// These names appear in the cmd/cover HTML interface.
		var longFile = path.Join(p.ImportPath, file)
		coverVars[file] = &FileVar{
			File: longFile,
			Var:  fmt.Sprintf("GoCover_%d_%x", coverIndex, h),
		}
		coverIndex++
	}

	return coverVars
}
