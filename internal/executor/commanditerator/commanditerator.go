package commanditerator

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
)

type CommandIterator struct {
	commands []*api.Command
	idx      int
}

func New(commands []*api.Command) *CommandIterator {
	return &CommandIterator{
		commands: commands,
		idx:      0,
	}
}

func (ci *CommandIterator) getOrPeek(failedAtLeastOnce bool, peek bool) *api.Command {
	shadowIdx := ci.idx
	defer func() {
		if peek {
			return
		}

		ci.idx = shadowIdx
	}()

	for {
		if shadowIdx >= len(ci.commands) {
			return nil
		}

		nextCommand := ci.commands[shadowIdx]

		shadowIdx++

		if shouldRun(nextCommand, failedAtLeastOnce) {
			return nextCommand
		}
	}
}

func (ci *CommandIterator) GetNext(failedAtLeastOnce bool) *api.Command {
	return ci.getOrPeek(failedAtLeastOnce, false)
}

func (ci *CommandIterator) PeekNext(failedAtLeastOnce bool) *api.Command {
	return ci.getOrPeek(failedAtLeastOnce, true)
}

func shouldRun(command *api.Command, failedAtLeastOnce bool) bool {
	return (command.ExecutionBehaviour == api.Command_ON_SUCCESS && !failedAtLeastOnce) ||
		(command.ExecutionBehaviour == api.Command_ON_FAILURE && failedAtLeastOnce) ||
		command.ExecutionBehaviour == api.Command_ALWAYS
}
