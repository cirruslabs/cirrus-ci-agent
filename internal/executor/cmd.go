//go:build !windows
// +build !windows

package executor

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/internal/shellwords"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func createCmd(scripts []string, customEnv *map[string]string) (*exec.Cmd, *os.File, error) {
	cmdShell := "/bin/sh"
	if bashPath, err := exec.LookPath("bash"); err == nil {
		cmdShell = bashPath
	}
	if customEnv != nil {
		if customShell, ok := (*customEnv)["CIRRUS_SHELL"]; ok {
			cmdShell = customShell
		}
	}

	if cmdShell == "direct" {
		cmdArgs := shellwords.ToArgv(ExpandText(scripts[0], *customEnv))
		if len(cmdArgs) > 1 {
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			return cmd, nil, nil
		}
		cmd := exec.Command(cmdArgs[0])
		return cmd, nil, nil
	}

	scriptFile, err := TempFileName("scripts", ".sh")
	if err != nil {
		return nil, nil, err
	}
	// add shebang
	scriptFile.WriteString(fmt.Sprintf("#!%s\n", cmdShell))
	scriptFile.WriteString("set -e\n")
	if strings.Contains(cmdShell, "bash") {
		scriptFile.WriteString("set -o pipefail\n")
	}
	scriptFile.WriteString("set -o verbose\n")
	for i := 0; i < len(scripts); i++ {
		scriptFile.WriteString(scripts[i])
		scriptFile.WriteString("\n")
	}
	scriptFile.Close()
	scriptFile.Chmod(os.FileMode(0777))
	cmd := exec.Command(cmdShell, scriptFile.Name())

	// Run CMD in it's own session
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	return cmd, scriptFile, nil
}
