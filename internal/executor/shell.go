package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type ShellOutputHandler func(bytes []byte) (int, error)

type ShellOutputWriter struct {
	io.Writer
	handler ShellOutputHandler
}

var TimeOutError = errors.New("timed out")

func (writer ShellOutputWriter) Write(bytes []byte) (int, error) {
	return writer.handler(bytes)
}

func ShellCommandsAndGetOutput(scripts []string, custom_env *map[string]string, timeout *<-chan time.Time) (bool, string) {
	var buffer bytes.Buffer
	cmd, err := ShellCommandsAndWait(scripts, custom_env, func(bytes []byte) (int, error) {
		return buffer.Write(bytes)
	}, timeout)
	return err == nil && cmd.ProcessState.Success(), buffer.String()
}

// return true if executed successful
func ShellCommandsAndWait(scripts []string, custom_env *map[string]string, handler ShellOutputHandler, executionTimeout *<-chan time.Time) (*exec.Cmd, error) {
	cmd, err := ShellCommands(scripts, custom_env, handler)
	if err != nil {
		return nil, err
	}

	done := make(chan error)
	go func() {
		// give time to flush logs
		time.Sleep(1 * time.Second)
		done <- cmd.Wait()
	}()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				done <- nil
			}
		}
	}()

	timeout := make(<-chan time.Time)
	if executionTimeout != nil {
		timeout = *executionTimeout
	}

	select {
	case <-timeout:
		handler([]byte("\nTimed out!"))
		err = cmd.Process.Kill()
		if err != nil {
			handler([]byte(fmt.Sprintf("\nFailed to gracefully kill: %s", err)))
		}
		err = cmd.Wait()
		if err != nil {
			handler([]byte(fmt.Sprintf("\nFailed to wait for killed shell: %s", err)))
		}
		return cmd, TimeOutError
	case <-done:
		if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				handler([]byte("\nSignaled to exit!"))
			}
			exitStatus := ws.ExitStatus()
			if exitStatus > 1 {
				handler([]byte(fmt.Sprintf("\nExit status: %d", exitStatus)))
			}
		} else {
			log.Printf("Failed to get wait status: %v", cmd.ProcessState.Sys())
		}
		return cmd, nil
	}
}

func ShellCommands(scripts []string, custom_env *map[string]string, handler ShellOutputHandler) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	var scriptFile *os.File
	var err error

	cmd, scriptFile, err = createCmd(scripts, custom_env)

	if scriptFile != nil {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			os.Remove(scriptFile.Name())
		}()
	}

	if err != nil {
		message := fmt.Sprintf("Error creating command-line script: %s", err)
		handler([]byte(message))
		return nil, errors.New(message)
	}

	env := os.Environ()
	if custom_env != nil {
		for k, v := range *custom_env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		if _, environmentAlreadyHasShell := os.LookupEnv("SHELL"); environmentAlreadyHasShell {
			_, userSpecifiedShell := (*custom_env)["SHELL"]
			if shellOverride, userSpecifiedCustomShell := (*custom_env)["CIRRUS_SHELL"]; userSpecifiedCustomShell && !userSpecifiedShell {
				env = append(env, fmt.Sprintf("SHELL=%s", shellOverride))
			}
		}
	}

	cmd.Env = env
	if custom_env != nil {
		if workingDir, ok := (*custom_env)["CIRRUS_WORKING_DIR"]; ok {
			EnsureFolderExists(workingDir)
			cmd.Dir = workingDir
		}
	}

	writer := ShellOutputWriter{
		handler: handler,
	}

	cmd.Stderr = &writer
	cmd.Stdout = &writer

	err = cmd.Start()
	if err != nil {
		message := fmt.Sprintf("Error starting command: %s", err)
		handler([]byte(message))
		return nil, errors.New(message)
	}

	return cmd, nil
}
