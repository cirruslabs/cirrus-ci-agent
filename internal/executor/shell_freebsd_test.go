// +build freebsd

package executor

import (
	"testing"
	"time"
)

func Test_Z_Shell(t *testing.T) {
	testEnv := map[string]string{}
	timeoutChan := time.After(time.Minute)
	_, output := ShellCommandsAndGetOutput([]string{
		"zsh -c 'echo \"foo:bar:baz\" | read -d \":\" line && echo $line'",
	}, &testEnv, &timeoutChan)

	if output == "foo\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}
