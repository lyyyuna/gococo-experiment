package compile

import "github.com/lyyyuna/gococo/pkg/log"

func (c *Compile) inject() {
	log.StartWait("injecting the coverage code")

	log.StopWait()
	log.Donef("coverage code injected")
}
