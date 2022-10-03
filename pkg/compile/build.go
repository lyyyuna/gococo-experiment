package compile

import (
	"os"
	"os/exec"
	"strings"

	"github.com/lyyyuna/gococo/pkg/log"
)

func (c *Compile) Build() {
	log.StartWait("building the injected project")

	args := []string{"build"}
	args = append(args, c.modifiedFlags...)
	args = append(args, c.modifedArgs...)
	// go build [-o output] [build flags] [packages]
	cmd := exec.Command("go", args...)
	cmd.Dir = c.tmpWd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Infof("go build cmd is: %v, in path [%v]", nicePrintArgs(cmd.Args), cmd.Dir)
	if err := cmd.Start(); err != nil {
		log.Fatalf("fail to execute go build: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatalf("fail to execute go build: %v", err)
	}

	// done
	log.StopWait()
	log.Donef("go build done")
}

// nicePrintArgs enhance display
//
//	 without nicePrintArgs, the output will be confusing:
//		 `go build -ldflags "-X my/package/config.Version=1.0.0" -o /home/lyy/gitdown/gin-test/cmd .`
//		 will be changed to
//		 `go build -ldflags -X my/package/config.Version=1.0.0 -o /home/lyy/gitdown/gin-test/cmd .`
func nicePrintArgs(args []string) []string {
	output := make([]string, 0)
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			output = append(output, "\""+arg+"\"")
		} else {
			output = append(output, arg)
		}
	}

	return output
}
