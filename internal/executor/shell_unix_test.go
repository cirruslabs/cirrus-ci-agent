//go:build !windows
// +build !windows

package executor

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestZshDoesNotHang(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("no Zsh found")
	}

	success, _ := ShellCommandsAndGetOutput(ctx, []string{"zsh -c 'echo \"a:b\" | read -d \":\" piece'"}, nil)
	assert.True(t, success)
}

func Test_ShellCommands_Unix(t *testing.T) {
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{"echo 'Foo'"}, nil)
	if output == "echo 'Foo'\nFoo\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Multiline_Unix(t *testing.T) {
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{"echo 'Foo'", "echo 'Bar'"}, nil)
	if output == "echo 'Foo'\nFoo\necho 'Bar'\nBar\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Fail_Fast_Unix(t *testing.T) {
	success, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"echo 'Hello!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, nil)
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
	testEnv := map[string]string{
		"FOO": "BAR",
	}
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"echo $FOO",
	}, &testEnv)

	if output == "echo $FOO\nBAR\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_CustomWorkingDir_Unix(t *testing.T) {
	testEnv := map[string]string{
		"CIRRUS_WORKING_DIR": "/tmp/cirrus-go-agent",
	}
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"pwd",
	}, &testEnv)

	expectedOutput := "pwd\n/tmp/cirrus-go-agent\n"

	if runtime.GOOS == "darwin" {
		expectedOutput = "pwd\n/private/tmp/cirrus-go-agent\n"
	}

	if output == expectedOutput {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func Test_ShellCommands_Timeout_Unix(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, output := ShellCommandsAndGetOutput(ctx, []string{"sleep 60"}, nil)
	if output == "sleep 60\n\nTimed out!" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}

func TestChildrenProcessesAreNotWaitedFor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	success, output := ShellCommandsAndGetOutput(ctx, []string{"sleep 60 &", "exit 0"}, nil)

	assert.True(t, success)
	assert.NotContains(t, output, "Timed out!")
	assert.NotContains(t, output, "error")
}
