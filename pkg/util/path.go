package util

import (
	"os"
	"path/filepath"
	"strings"
)

// SubElem checks if `sub` is in the `parent` directory
func SubElem(parent, sub string) bool {
	up := ".." + string(os.PathSeparator)

	// path-comparisons using filepath.Abs don't work reliably
	rel, err := filepath.Rel(parent, sub)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(rel, up) && rel != ".." {
		return true
	}
	return false
}
