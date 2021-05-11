// +build !windows

package executor

import (
	"os/exec"
	"syscall"
)

type ShellCommands struct {
	cmd *exec.Cmd
}

func (sc *ShellCommands) afterStart() {
	// only used on Windows
}

func (sc *ShellCommands) kill() error {
	return syscall.Kill(-sc.cmd.Process.Pid, syscall.SIGKILL)
}
