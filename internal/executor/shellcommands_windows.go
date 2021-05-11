package executor

import (
	"golang.org/x/sys/windows"
	"os/exec"
)

type ShellCommands struct{
	cmd *exec.Cmd
	jobHandle windows.Handle
}

func (sc *ShellCommands) afterStart() {
	jobHandle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	sc.jobHandle = jobHandle

	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA | windows.PROCESS_TERMINATE,
		false, uint32(sc.cmd.Process.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(process)

	if err := windows.AssignProcessToJobObject(jobHandle, process); err != nil {
		return
	}
}

func (sc *ShellCommands) kill() error {
	if sc.jobHandle == 0 {
		return sc.cmd.Process.Kill()
	}

	if err := windows.TerminateJobObject(sc.jobHandle, 0); err != nil {
		return err
	}

	return windows.CloseHandle(sc.jobHandle)
}
