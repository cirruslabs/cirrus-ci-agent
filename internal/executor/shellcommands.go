//go:build !windows
// +build !windows

package executor

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/piper"
	"os/exec"
	"syscall"
)

type ShellCommands struct {
	cmd   *exec.Cmd
	piper *piper.Piper
}

func (sc *ShellCommands) afterStart() error {
	// only used on Windows

	return nil
}

func (sc *ShellCommands) kill() error {
	return syscall.Kill(-sc.cmd.Process.Pid, syscall.SIGKILL)
}
