// +build !windows

package executor

import (
	"context"
	"github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// TestProcessGroupTermination ensures that we terminate the whole process group that
// the shell spawned in ShellCommandsAndGetOutput() has been placed into, thus killing
// it's children processes.
func TestProcessGroupTermination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	success, output := ShellCommandsAndGetOutput(ctx, []string{"sleep 86400 & echo target PID is $! ; sleep 60"}, nil)

	assert.False(t, success, "the command should fail due to time out error")
	assert.Contains(t, output, "Timed out!", "the command should time out")

	re := regexp.MustCompile(".*target PID is ([0-9]+).*")
	matches := re.FindStringSubmatch(output)
	if len(matches) != 2 {
		t.Fatal("failed to find target PID")
	}

	pid, err := strconv.ParseInt(matches[1], 10, 32)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the zombie to be reaped by the init process
	time.Sleep(5 * time.Second)

	// Unfortunately go-ps error behavior differs depending on the OS,
	// (missing process is an error on FreeBSD but there's no error on Linux),
	// so we skip the check here
	process, _ := ps.FindProcess(int(pid))
	assert.Nil(t, process)
}

func TestZshDoesNotHang(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	success, output := ShellCommandsAndGetOutput(ctx, []string{"sleep 60 & sleep 1"}, nil)

	assert.True(t, success)
	assert.NotContains(t, output, "Timed out!")
}
