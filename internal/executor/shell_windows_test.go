// +build windows

package executor

import (
	"testing"
)

func Test_ShellCommands_Windows(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	}
	_, output := ShellCommandsAndGetOutput([]string{"echo 'Foo'"}, &test_env, nil)
	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Foo' \r\n'Foo'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Multiline_Windows(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	}
	_, output := ShellCommandsAndGetOutput([]string{"echo 'Foo'", "echo 'Bar'"}, &test_env, nil)
	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Foo' \r\n'Foo'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call echo 'Bar' \r\n'Bar'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Fail_Fast_Windows(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	}
	success, output := ShellCommandsAndGetOutput([]string{
		"echo 'Hello!'",
		"echo 'Friend!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, &test_env, nil)
	if success {
		t.Error("Should fail!")
	}

	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Hello!' \r\n'Hello!'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call echo 'Friend!' \r\n'Friend!'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call exit 1 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Environment_Windows(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
		"FOO":                "BAR",
	}
	_, output := ShellCommandsAndGetOutput([]string{
		"echo %FOO%",
	}, &test_env, nil)

	expected_output := "\r\nC:\\Windows\\TEMP>call echo BAR \r\nBAR\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_Exit_Code_Windows(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	}
	success, output := ShellCommandsAndGetOutput([]string{
		"export FOO=239",
		"echo %ERRORLEVEL%",
		"echo 'Unreachable!'",
	}, &test_env, nil)

	if success {
		t.Errorf("Should've failed! '%+q'", output)
	}

	expected_output := "\r\nC:\\Windows\\TEMP>call export FOO=239 \r\n'export' is not recognized as an internal or external command,\r\noperable program or batch file.\r\n\r\nC:\\Windows\\TEMP>if 1 NEQ 0 exit /b 1 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_Powershell(t *testing.T) {
	test_env := map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
		"CIRRUS_SHELL":       "powershell",
	}
	success, output := ShellCommandsAndGetOutput([]string{
		"echo 'Foo!'",
		"echo 'Bar!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, &test_env, nil)

	if success {
		t.Errorf("Should've fail! '%+q'", output)
	}

	expected_output := "Foo!\r\nBar!\r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}
