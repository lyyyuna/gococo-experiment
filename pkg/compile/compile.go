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

	// isBuildModVendor
	isBuildModVendor bool

	// curWd repesents the current working directory
	curWd string

	// curProjectRootDir repesents the current project root directory
	curProjectRootDir string

	// curGoWork represents the go.work file path if exists
	curGoWork string

	// projectModulePath represents the [module-path] of the project
	projectModulePath string

	// pkgs
	pkgs map[string]*Package
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
func WithArgs(args ...string) Option {
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

	// lock coping + injecting
	compileLock := newCompileMutex(filepath.Join(c.curProjectRootDir, ".gococo.lock"), time.Second*360)
	if err := compileLock.Lock(); err != nil {
		log.Fatalf("fail to lock the project: %v", err)
	}
	defer compileLock.Unlock()

	return c
}
