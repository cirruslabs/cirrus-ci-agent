//go:build !(openbsd || netbsd)

package processdumper

import (
	"github.com/mitchellh/go-ps"
	gopsutilprocess "github.com/shirou/gopsutil/process"
	"log"
)

func Dump() {
	processes, err := ps.Processes()
	if err != nil {
		log.Printf("Failed to retrieve processes to diagnose the time out")
	} else {
		log.Printf("Process list:")
		for _, process := range processes {
			log.Printf("%d %d %s", process.Pid(), process.PPid(), processExeOrCmdline(process))
		}
	}
}

func processExeOrCmdline(process ps.Process) string {
	gopsutilProcess, err := gopsutilprocess.NewProcess(int32(process.Pid()))
	if err != nil {
		// Fall back to just the comm value
		return process.Executable()
	}

	cmdline, err := gopsutilProcess.Cmdline()
	if err != nil {
		// Fall back to just the comm value
		return process.Executable()
	}

	return cmdline
}
