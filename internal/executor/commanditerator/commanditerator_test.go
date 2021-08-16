package commanditerator_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/commanditerator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptyCommandList(t *testing.T) {
	ci := commanditerator.New([]*api.Command{})

	assert.Nil(t, ci.PeekNext(false))
	assert.Nil(t, ci.GetNext(false))
}

func TestPeek(t *testing.T) {
	ci := commanditerator.New([]*api.Command{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	})

	assert.Equal(t, "first", ci.PeekNext(false).Name)
	assert.Equal(t, "first", ci.GetNext(false).Name)
	assert.Equal(t, "second", ci.PeekNext(false).Name)
	assert.Equal(t, "second", ci.GetNext(false).Name)
	assert.Equal(t, "third", ci.PeekNext(false).Name)
	assert.Equal(t, "third", ci.GetNext(false).Name)
	assert.Nil(t, ci.PeekNext(false))
	assert.Nil(t, ci.GetNext(false))
}

func TestPeekIsPure(t *testing.T) {
	ci := commanditerator.New([]*api.Command{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	})

	assert.Equal(t, "first", ci.PeekNext(false).Name)
	assert.Equal(t, "first", ci.PeekNext(false).Name)
}

func TestExecutionBehaviorIsRespected(t *testing.T) {
	ci := commanditerator.New([]*api.Command{
		{Name: "should be skipped"},
		{Name: "should be returned", ExecutionBehaviour: api.Command_ALWAYS},
	})

	assert.Equal(t, "should be returned", ci.PeekNext(true).Name)
	assert.Equal(t, "should be returned", ci.GetNext(true).Name)
}
