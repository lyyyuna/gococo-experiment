package compile

import (
	"flag"
	"path/filepath"

	"github.com/lyyyuna/gococo/pkg/log"
)

// parseArgs parses original command line args
func (c *Compile) parseArgs() {
	var goflags goConfig

	addBuildFlags := func(cmdSet *flag.FlagSet) {
		cmdSet.BoolVar(&goflags.BuildA, "a", false, "")
		cmdSet.BoolVar(&goflags.BuildN, "n", false, "")
		cmdSet.IntVar(&goflags.BuildP, "p", 4, "")
		cmdSet.BoolVar(&goflags.BuildV, "v", false, "")
		cmdSet.BoolVar(&goflags.BuildX, "x", false, "")
		cmdSet.StringVar(&goflags.BuildBuildmode, "buildmode", "default", "")
		cmdSet.StringVar(&goflags.BuildMod, "mod", "", "")
		cmdSet.StringVar(&goflags.Installsuffix, "installsuffix", "", "")

		// well, the variable type is different from the official "go comand" flags,
		// I just use them to consume the command line...
		cmdSet.StringVar(&goflags.BuildAsmflags, "asmflags", "", "")
		cmdSet.StringVar(&goflags.BuildCompiler, "compiler", "", "")
		cmdSet.StringVar(&goflags.BuildGcflags, "gcflags", "", "")
		cmdSet.StringVar(&goflags.BuildGccgoflags, "gccgoflags", "", "")
		// mod related
		cmdSet.BoolVar(&goflags.ModCacheRW, "modcacherw", false, "")
		cmdSet.StringVar(&goflags.ModFile, "modfile", "", "")
		cmdSet.StringVar(&goflags.BuildLdflags, "ldflags", "", "")
		cmdSet.BoolVar(&goflags.BuildLinkshared, "linkshared", false, "")
		cmdSet.StringVar(&goflags.BuildPkgdir, "pkgdir", "", "")
		cmdSet.BoolVar(&goflags.BuildRace, "race", false, "")
		cmdSet.BoolVar(&goflags.BuildMSan, "msan", false, "")
		cmdSet.StringVar(&goflags.BuildTags, "tags", "", "")
		cmdSet.StringVar(&goflags.BuildToolexec, "toolexec", "", "")
		cmdSet.BoolVar(&goflags.BuildTrimpath, "trimpath", false, "")
		cmdSet.BoolVar(&goflags.BuildWork, "work", false, "")
	}

	addOutputFlags := func(cmdSet *flag.FlagSet) {
		cmdSet.StringVar(&goflags.BuildO, "o", "", "")
	}

	goFlagSets := flag.NewFlagSet("GO jiayi shi tiancai !!!", flag.ContinueOnError)
	addBuildFlags(goFlagSets)
	addOutputFlags(goFlagSets)
	err := goFlagSets.Parse(c.oriArgs)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// check if -o is set
	var oset bool
	flags := make([]string, 0)
	goFlagSets.Visit(func(f *flag.Flag) {
		// turn the output dir to absolute path, as we will do compile in a temporary dir
		if f.Name == "o" {
			outputDir := f.Value.String()
			outputDir, err := filepath.Abs(outputDir)
			if err != nil {
				log.Fatalf("output flag is not valid: %v", err)
			}
			flags = append(flags, "-o", outputDir)
			oset = true
		} else {
			flags = append(flags, "-"+f.Name, f.Value.String())
		}
	})

	// if -o is not set, output the binary to the orignal working directory
	if !oset && c.compileType == GOCOCO_DO_BUILD {
		flags = append(flags, "-o", c.curWd)
	}

	c.modifiedFlags = flags
	c.modifedArgs = goFlagSets.Args()
	c.buildTags = goflags.BuildTags
	c.buildMod = goflags.BuildMod
}

type goConfig struct {
	BuildA                 bool
	BuildBuildmode         string // -buildmode flag
	BuildMod               string // -mod flag
	BuildModReason         string // reason -mod flag is set, if set by default
	BuildI                 bool   // -i flag
	BuildLinkshared        bool   // -linkshared flag
	BuildMSan              bool   // -msan flag
	BuildN                 bool   // -n flag
	BuildO                 string // -o flag
	BuildP                 int    // -p flag
	BuildPkgdir            string // -pkgdir flag
	BuildRace              bool   // -race flag
	BuildToolexec          string // -toolexec flag
	BuildToolchainName     string
	BuildToolchainCompiler func() string
	BuildToolchainLinker   func() string
	BuildTrimpath          bool // -trimpath flag
	BuildV                 bool // -v flag
	BuildWork              bool // -work flag
	BuildX                 bool // -x flag
	// from buildcontext
	Installsuffix string // -installSuffix
	BuildTags     string // -tags
	// from load
	BuildAsmflags   string
	BuildCompiler   string
	BuildGcflags    string
	BuildGccgoflags string
	BuildLdflags    string

	// mod related
	ModCacheRW bool
	ModFile    string
}

func (c *Compile) getPackages() {

}
