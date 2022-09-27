package compile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/lyyyuna/gococo/pkg/log"
)

const (
	CACHE_ROOT_DIR = ".gococo"
	CACHE_DIGEST   = "digest.modtime"
)

// cache can skip coping to temp if files not changed.
//
// the cache layout:
//
//	.gococo				 // cacheRootDir
//	  ├─ project            // cacheDir
//	  └─ digest.modtime  // digest file
type cache struct {
	// the path for the digest file
	digestFilePath string

	// the digest info of last time
	oldDigest map[string]int64

	// the digest info of this time
	newDigest map[string]int64

	// tell if the cache needs to refresh
	needsRefresh bool

	// the base directory of target
	targetDir string

	// the corresponding target directory in the cache
	cacheDir string

	// cacheRootDir
	cacheRootDir string

	// to skip some files, like `.git` and self
	skipPattern map[string]struct{}

	// package information of the project
	pkgs []*Package
}

type cacheOption func(*cache)

func withSkip(p string) cacheOption {
	return func(bc *cache) {
		skipPath := filepath.Join(bc.targetDir, p)
		bc.skipPattern[skipPath] = struct{}{}
	}
}

func withPackage(pkgs map[string]*Package) cacheOption {
	return func(bc *cache) {
		for _, pkg := range pkgs {
			bc.pkgs = append(bc.pkgs, pkg)
		}
	}
}

func newCache(target string, opts ...cacheOption) *cache {
	if target == "" {
		log.Fatalf("empty target for the cache")
	}

	bc := &cache{
		oldDigest:   make(map[string]int64),
		newDigest:   make(map[string]int64),
		targetDir:   target,
		skipPattern: make(map[string]struct{}),
		pkgs:        make([]*Package, 0),
	}

	for _, o := range opts {
		o(bc)
	}

	if dir := os.Getenv("GOCOCO_CACHE_DIR"); dir != "" {
		bc.cacheRootDir = filepath.Join(target, dir)
	} else {
		bc.cacheRootDir = filepath.Join(target, CACHE_ROOT_DIR)
	}

	bc.cacheDir = filepath.Join(bc.cacheRootDir, "porject")
	bc.digestFilePath = filepath.Join(bc.cacheRootDir, CACHE_DIGEST)

	// skip self
	bc.skipPattern[bc.cacheRootDir] = struct{}{}

	// load old digest from cache
	if found := bc.loadOldDigest(); !found {
		bc.needsRefresh = true
	}

	return bc
}

// Refreshed tells if the cache is refreshed
func (bc *cache) Refreshed() bool {
	return bc.needsRefresh
}

func (bc *cache) doCopy() {
	// get new digest from target
	bc.getNewDigest()

	// check if need to refresh cache
	if !bc.needsRefresh {
		eq := reflect.DeepEqual(bc.newDigest, bc.oldDigest)
		if eq {
			bc.needsRefresh = false
			return
		} else {
			bc.needsRefresh = true
		}
	}

	// remove old cache
	if err := os.RemoveAll(bc.cacheRootDir); err != nil {
		log.Fatalf("fail to remove old cache: %v", err)
	}

	// create new cache dir
	if err := os.MkdirAll(bc.cacheRootDir, os.ModePerm); err != nil {
		log.Fatalf("fail to make cache: %v", err)
	}

	// copy all files
	bc.doRealCopy()
}

func (bc *cache) loadOldDigest() (found bool) {
	_, err := os.Lstat(bc.digestFilePath)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		log.Fatalf("fail to locate the digest info: %v", err)
	} else {
		f, err := os.Open(bc.digestFilePath)
		if err != nil {
			log.Fatalf("fail to open load the digest info: %v", err)
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			trimed := strings.TrimSpace(s.Text())
			if len(trimed) == 0 {
				continue
			}
			line := strings.Split(trimed, " ")
			if len(line) != 2 {
				log.Fatalf("the line in digest file is in wrong format: %v", trimed)
			}
			modTime, err := strconv.ParseInt(line[1], 10, 64)
			if err != nil {
				log.Fatalf("fail to parse digest info of: %v", trimed)
			}
			bc.oldDigest[line[0]] = modTime
		}
	}

	return true
}

func (bc *cache) getNewDigest() {
	// find all source files
	srcFiles := make([]string, 0)
	for _, pkg := range bc.pkgs {
		srcFiles = append(srcFiles, bc.sourceFiles(pkg)...)
	}

	for _, src := range srcFiles {
		info, err := os.Lstat(src)
		if err != nil {
			log.Fatalf("fail to get %v's info: %v", src, err)
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			log.Debugf("found symlink: %v, follow the symlink to check mod time", src)
			orig, err := os.Readlink(src)
			if err != nil {
				log.Fatalf("fail to read symlink: %v", err)
			}

			f, err := os.Stat(orig)
			if err != nil {
				log.Fatalf("fail to get %v's info: %v", src, err)
			}

			bc.newDigest[src] = f.ModTime().UnixNano()

		default:
			f, err := os.Stat(src)
			if err != nil {
				log.Fatalf("fail to get %v's info: %v", src, err)
			}

			bc.newDigest[src] = f.ModTime().UnixNano()
		}
	}
}

func (bc *cache) doRealCopy() {
	srcFiles := make([]string, 0)
	modFile := ""
	for _, pkg := range bc.pkgs {
		if modFile == "" {
			modFile = pkg.Module.GoMod
			sumFile := filepath.Join(bc.targetDir, "go.sum")
			srcFiles = append(srcFiles, modFile, sumFile)
		}
		srcFiles = append(srcFiles, bc.sourceFiles(pkg)...)
	}

	for _, src := range srcFiles {
		relPath, err := filepath.Rel(bc.targetDir, src)
		if err != nil {
			log.Fatalf("the file: %v is not in the project directory, gococo currently cannot deal with such file", src)
		}

		dst := filepath.Join(bc.cacheDir, relPath)
		dstDir := filepath.Dir(dst)
		err = os.MkdirAll(dstDir, os.ModePerm)
		if err != nil {
			log.Fatalf("fail to create the directory in the cache : %v, %v", dstDir, err)
		}

		f, err := os.Create(dst)
		if err != nil {
			log.Fatalf("fail to create the file in the cache directory: %v", err)
		}
		defer f.Close()

		s, err := os.Open(src)
		if err != nil {
			log.Fatalf("fail to open the original file: %v", err)
		}
		defer s.Close()

		if _, err = io.Copy(f, s); err != nil {
			log.Fatalf("fail to copy the file: %v", err)
		}
	}
}

// saveDigest saves the digest info to the disk
func (bc *cache) saveDigest() {
	f, err := os.Create(bc.digestFilePath)
	if err != nil {
		log.Fatalf("fail to create the new digest file: %v", err)
	}
	defer f.Close()

	for path, modTime := range bc.newDigest {
		line := fmt.Sprintf("%v %v\n", path, modTime)
		f.WriteString(line)
	}
}

func (bc *cache) sourceFiles(pkg *Package) []string {
	out := make([]string, 0)
	base := pkg.Dir

	help := func(s string) string {
		return filepath.Join(base, s)
	}

	for _, f := range pkg.GoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CgoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CompiledGoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.IgnoredGoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.IgnoredOtherFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.CXXFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.MFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.HFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.FFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SwigFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SwigCXXFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.SysoFiles {
		out = append(out, help(f))
	}

	for _, f := range pkg.EmbedFiles {
		out = append(out, help(f))
	}

	return out
}
