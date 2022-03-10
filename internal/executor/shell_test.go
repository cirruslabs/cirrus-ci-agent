package executor

import (
	"bytes"
	"context"
)

func ShellCommandsAndGetOutput(
	ctx context.Context,
	scripts []string,
	custom_env *map[string]string,
) (bool, string) {
	var buffer bytes.Buffer
	cmd, err := ShellCommandsAndWait(ctx, scripts, custom_env, func(bytes []byte) (int, error) {
		return buffer.Write(bytes)
	}, false)
	return err == nil && cmd.ProcessState.Success(), buffer.String()
}
