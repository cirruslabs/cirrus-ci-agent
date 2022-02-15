//go:build !(openbsd || netbsd)

package processdumper

import (
	"github.com/mitchellh/go-ps"
	"log"
)

func Dump() {
	processes, err := ps.Processes()
	if err != nil {
		log.Printf("Failed to retrieve processes to diagnose the time out")
	} else {
		log.Printf("Process list:")
		for _, process := range processes {
			log.Printf("%d %d %s", process.Pid(), process.PPid(), process.Executable())
		}
	}
}
