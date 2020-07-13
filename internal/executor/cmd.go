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

func createCmd(scripts []string, custom_env *map[string]string) (*exec.Cmd, *os.File, error) {
	cmdShell := "/bin/sh"
	if bashPath, err := exec.LookPath("bash"); err == nil {
		cmdShell = bashPath
	}
	if custom_env != nil {
		if customShell, ok := (*custom_env)["CIRRUS_SHELL"]; ok {
			cmdShell = customShell
		}
	}

	if cmdShell == "direct" {
		cmdArgs := shellwords.ToArgv(ExpandText(scripts[0], *custom_env))
		if len(cmdArgs) > 1 {
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			return cmd, nil, nil
		} else {
			cmd := exec.Command(cmdArgs[0])
			return cmd, nil, nil
		}
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

	// run CMD in it's own group
	// https://stackoverflow.com/questions/33165530/prevent-ctrlc-from-interrupting-exec-command-in-golang
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd, scriptFile, nil
}
