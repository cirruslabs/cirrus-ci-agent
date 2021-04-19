package executor

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLimitCommands(t *testing.T) {
	commands := []*api.Command{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
		{Name: "d"},
	}

	examples := []struct {
		Description      string
		FromName, ToName string
		Expected         []*api.Command
	}{
		{"unspecified bounds", "", "", commands},
		{"zero bound (beginning)", "a", "a", []*api.Command{}},
		{"zero bound (middle)", "b", "b", []*api.Command{}},
		{"zero bound (ending)", "d", "d", []*api.Command{}},
		{"zero bound (unspecified beginning)", "", "a", []*api.Command{}},
		{"only from (beginning)", "a", "", commands},
		{"only from (middle)", "b", "", []*api.Command{
			{Name: "b"},
			{Name: "c"},
			{Name: "d"},
		}},
		{"only from (ending)", "d", "", []*api.Command{
			{Name: "d"},
		}},
		{"only to (beginning)", "", "b", []*api.Command{
			{Name: "a"},
		}},
		{"only to (middle)", "", "c", []*api.Command{
			{Name: "a"},
			{Name: "b"},
		}},
		{"only to (ending)", "", "d", []*api.Command{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		}},
		{"nonexistent", "X", "Y", commands},
	}

	for _, example := range examples {
		t.Run(example.Description, func(t *testing.T) {
			require.Equal(t, example.Expected, BoundedCommands(commands, example.FromName, example.ToName))
		})
	}
}
