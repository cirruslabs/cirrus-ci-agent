package commanditerator_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/commanditerator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptyCommandList(t *testing.T) {
	ci := commanditerator.New([]*api.Command{})

	assert.Nil(t, ci.GetNext(false))
}

func TestPeek(t *testing.T) {
	ci := commanditerator.New([]*api.Command{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	})

	assert.Equal(t, "first", ci.GetNext(false).Name)
	assert.Equal(t, "second", ci.GetNext(false).Name)
	assert.Equal(t, "third", ci.GetNext(false).Name)
	assert.Nil(t, ci.GetNext(false))
}

func TestExecutionBehaviorIsRespected(t *testing.T) {
	var commands = []*api.Command{
		{Name: "should be skipped"},
		{Name: "should be returned", ExecutionBehaviour: api.Command_ALWAYS},
	}

	ci := commanditerator.New(commands)
	next, skipped := ci.GetNextWithSkipped(true)
	assert.Equal(t, "should be skipped", next.Name)
	assert.True(t, skipped)

	ci = commanditerator.New(commands)
	assert.Equal(t, "should be returned", ci.GetNext(true).Name)
}
