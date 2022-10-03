package compile

import (
	"os"
	"path/filepath"
	"time"

	"github.com/lyyyuna/gococo/pkg/log"
)

const (
	GOCOCO_DO_BUILD = iota
	GOCOCO_DO_INSTALL
	GOCOCO_DO_RUN
)

// Compile is just that, a compile + code injection for your program.
//
// The compile type can be either:
//  1. build
//  2. install
//  3. run
type Compile struct {
	// compileType is the compile type
	//  1. GOCOCO_DO_BUILD
	//  2. GOCOCO_DO_INSTALL
	//  3. GOCOCO_DO_RUN
	compileType int

	// oriArgs represents the original arguments + flags in the command line
	oriArgs []string

	// modifiedFlags, is that we will change some flag content, like -o
	modifiedFlags []string

	// modifedArgs delete flags from oriArgs, leave only pure arguments
	modifedArgs []string

	// buildTags, extract go build tags from the original args, as they will change the compile behavior
	buildTags string

	// buildMod, extract go build mod type from the original args
	buildMod string

	// curWd repesents the current working directory
	curWd string

	// curProjectRootDir repesents the current project root directory,
	// it may be root of go.mod or root of go.work
	curProjectRootDir string

	// envGOWORK represents the go.work file path from go env
	envGOWORK string

	// envGOMOD represents the cloesest go.mod file of current directory if exists
	envGOMOD string

	// isGoWork tells if the project is workspace
	isGoWork bool

	// modulePaths
	modulePaths map[string]*modProject

	// tmpWd represents the corresponding working directory in the cache
	tmpWd string

	// tmpProjectRootDir represents the corresponding project directory in the cache
	tmpProjectRootDir string

	// pkgs
	pkgs map[string]*Package
}

// modProject represents each go.mod sub-project in the root project
type modProject struct {
	// path is the absolute path of the mod project
	path string

	// tmpPath is the absolute path of the temporary mod project
	tmpPath string

	// gomodPath is the absolute path of the go.mod file
	gomodPath string

	// tmpGomodPath is the absolute path of the go.mod file in the temporary project
	tmpGomodPath string

	// inRootProject
	//  ``` go.work
	//  for example:
	//  use (
	//    ./example
	//    ./hello
	//    /home/lyy/gitup/mygococo
	//  )
	//  ```
	//  the `/home/lyy/github/mygococo` here is not in the go.work's directory,
	//  so inRootProject=false
	inRootProject bool

	// modulePath is the [module-path] of the mod project
	modulePath string

	// isVendor tells if the project is in vendor mod
	isVendor bool
}

// Option represents a compile option
type Option func(*Compile)

// WithBuild, the compile type is build
func WithBuild() Option {
	return func(c *Compile) {
		c.compileType = GOCOCO_DO_BUILD
	}
}

// WithInstall, the compile type is install.
func WithInstall() Option {
	return func(c *Compile) {
		c.compileType = GOCOCO_DO_INSTALL
	}
}

// WithRun, the compile type is run.
func WithRun() Option {
	return func(c *Compile) {
		c.compileType = GOCOCO_DO_RUN
	}
}

// WithArgs specifies the original compile arguments.
func WithArgs(args []string) Option {
	return func(c *Compile) {
		c.oriArgs = append(c.oriArgs, args...)
	}
}

// NewCompile creates a new compile object, and do some initialization work...
func NewCompile(opts ...Option) *Compile {
	c := &Compile{
		oriArgs: make([]string, 0),
	}

	for _, o := range opts {
		o(c)
	}

	// we should get wd first!!!
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get current working directory: %v", err)
	}
	c.curWd = wd

	// parse the flags and args
	c.parseArgs()

	// get project meta info
	c.readProjectMetaInfo()
	log.Donef("project meta information parsed")
	c.displayProjectMetaInfo()

	// lock coping + injecting
	compileLock := newCompileMutex(filepath.Join(c.curProjectRootDir, ".gococo.lock"), time.Second*360)
	if err := compileLock.Lock(); err != nil {
		log.Fatalf("fail to lock the project: %v", err)
	}
	defer compileLock.Unlock()

	// copy the project to the temporary directory
	cache := newCache(c.curProjectRootDir,
		withPackage(c.pkgs),
		withImportPaths(c.modulePaths),
	)
	cache.doCopy()
	cache.saveDigest()
	// check if the cache is refreshed
	if cache.Refreshed() {
		log.Donef("project copied to the temporary directory")
	} else {
		log.Donef("no need to copy, using cached project")
		return c
	}

	// get tmp project meta info
	c.transformMetaInfoInCache(cache.GetProjectDir())

	// update the go mod file
	c.updateGoModGoWorkFile()

	return c
}
