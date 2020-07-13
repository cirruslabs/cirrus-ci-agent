// +build !windows

package executor

import (
	"runtime"
	"testing"
)

func Test_ShellCommands_Unix(t *testing.T) {
	_, output := ShellCommandsAndGetOutput([]string{"echo 'Foo'"}, nil, nil)
	if output == "echo 'Foo'\nFoo\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Multiline_Unix(t *testing.T) {
	_, output := ShellCommandsAndGetOutput([]string{"echo 'Foo'", "echo 'Bar'"}, nil, nil)
	if output == "echo 'Foo'\nFoo\necho 'Bar'\nBar\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Fail_Fast_Unix(t *testing.T) {
	success, output := ShellCommandsAndGetOutput([]string{
		"echo 'Hello!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, nil, nil)
	if success {
		t.Error("Should fail!")
	}

	if output == "echo 'Hello!'\nHello!\nexit 1\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Environment_Unix(t *testing.T) {
	test_env := map[string]string{
		"FOO": "BAR",
	}
	_, output := ShellCommandsAndGetOutput([]string{
		"echo $FOO",
	}, &test_env, nil)

	if output == "echo $FOO\nBAR\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_CustomWorkingDir_Unix(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "/tmp/cirrus-go-agent",
	}
	_, output := ShellCommandsAndGetOutput([]string{
		"pwd",
	}, &test_env, nil)

	expected_output := "pwd\n/tmp/cirrus-go-agent\n"

	if runtime.GOOS == "darwin" {
		expected_output = "pwd\n/private/tmp/cirrus-go-agent\n"
	}

	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}
