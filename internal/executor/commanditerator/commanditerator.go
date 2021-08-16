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

func (ci *CommandIterator) getOrPeek(
	failedAtLeastOnce bool,
	peek bool,
	includeSkipped bool,
) (command *api.Command, skipped bool) {
	shadowIdx := ci.idx
	defer func() {
		if peek {
			return
		}

		ci.idx = shadowIdx
	}()

	for {
		if shadowIdx >= len(ci.commands) {
			return nil, false
		}

		nextCommand := ci.commands[shadowIdx]

		shadowIdx++

		if shouldRun(nextCommand, failedAtLeastOnce) {
			return nextCommand, false
		} else if includeSkipped {
			return nextCommand, true
		}
	}
}

func (ci *CommandIterator) GetNext(failedAtLeastOnce bool) *api.Command {
	command, _ := ci.getOrPeek(failedAtLeastOnce, false, false)
	return command
}

func (ci *CommandIterator) GetNextWithSkipped(failedAtLeastOnce bool) (*api.Command, bool) {
	return ci.getOrPeek(failedAtLeastOnce, false, true)
}

func (ci *CommandIterator) PeekNext(failedAtLeastOnce bool) *api.Command {
	command, _ := ci.getOrPeek(failedAtLeastOnce, true, false)
	return command
}

func (ci *CommandIterator) PeekNextWithSkipped(failedAtLeastOnce bool) (*api.Command, bool) {
	return ci.getOrPeek(failedAtLeastOnce, true, true)
}

func shouldRun(command *api.Command, failedAtLeastOnce bool) bool {
	return (command.ExecutionBehaviour == api.Command_ON_SUCCESS && !failedAtLeastOnce) ||
		(command.ExecutionBehaviour == api.Command_ON_FAILURE && failedAtLeastOnce) ||
		command.ExecutionBehaviour == api.Command_ALWAYS
}
