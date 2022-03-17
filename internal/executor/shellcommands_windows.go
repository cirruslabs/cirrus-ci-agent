package executor

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/piper"
	"golang.org/x/sys/windows"
	"os/exec"
	"unsafe"
)

type ShellCommands struct {
	cmd       *exec.Cmd
	piper     *piper.Piper
	jobHandle windows.Handle
}

func (sc *ShellCommands) afterStart() error {
	jobHandle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	sc.jobHandle = jobHandle

	// Allow job explicit job breakaway
	basicLimitInformation := windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
		LimitFlags: windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK,
	}

	_, err = windows.SetInformationJobObject(jobHandle, windows.JobObjectBasicLimitInformation,
		uintptr(unsafe.Pointer(&basicLimitInformation)),
		uint32(unsafe.Sizeof(basicLimitInformation)))
	if err != nil {
		return err
	}

	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false, uint32(sc.cmd.Process.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)

	return windows.AssignProcessToJobObject(jobHandle, process)
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
