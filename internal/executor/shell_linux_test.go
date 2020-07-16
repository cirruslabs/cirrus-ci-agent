// +build !windows !freebsd

package executor

import (
	"testing"
)

func Test_DirectShell_Unix(t *testing.T) {
	testEnv := map[string]string{
		"CIRRUS_SHELL": "direct",
	}
	_, output := ShellCommandsAndGetOutput([]string{
		"bash -c 'echo $CIRRUS_SHELL'",
	}, &testEnv, nil)

	if output == "direct\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}
