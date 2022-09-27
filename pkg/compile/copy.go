package compile

import "github.com/lyyyuna/gococo/pkg/log"

// copyProject copies the original project to the temporary directory
func (c *Compile) copyProject() {
	log.StartWait("coping project to the temporary directory")

	buildCache := newCache(c.curProjectRootDir,
		withPackage(c.pkgs),
	)

	buildCache.doCopy()
}
