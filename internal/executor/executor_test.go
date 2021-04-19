package executor

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
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

func TestPopulateCloneAndWorkingDirEnvironmentVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
		return
	}
	tmpDirToRestore := os.Getenv("TMPDIR")
	_ = os.Setenv("TMPDIR", "/tmp")
	e := &Executor{}
	ePreCreate := &Executor{}
	ePreCreate.SetPreCreatedWorkingDir("/tmp/precreated-build")
	examples := []struct {
		Executor        *Executor
		Description     string
		Given, Expected map[string]string
	}{
		{
			e,
			"empty",
			map[string]string{},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/cirrus-ci-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			ePreCreate,
			"empty (precreated)",
			map[string]string{},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			e,
			"only working",
			map[string]string{
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "$CIRRUS_WORKING_DIR",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
		},
		{
			ePreCreate,
			"only working (precreated)",
			map[string]string{
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			e,
			"only working (monorepo)",
			map[string]string{
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/cirrus-ci-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/foo",
			},
		},
		{
			ePreCreate,
			"only working (monorepo + precreated)",
			map[string]string{
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/foo",
			},
		},
		{
			e,
			"only clone",
			map[string]string{
				"CIRRUS_CLONE_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			ePreCreate,
			"only clone (precreated)",
			map[string]string{
				"CIRRUS_CLONE_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			e,
			"both",
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
		},
		{
			ePreCreate,
			"both (precreated)",
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR",
			},
		},
		{
			e,
			"both (monorepo)",
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/bar",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/bar",
			},
		},
		{
			ePreCreate,
			"both (monorepo + precreated)",
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/bar",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/precreated-build",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/bar",
			},
		},
	}

	for _, example := range examples {
		t.Run(example.Description, func(t *testing.T) {
			require.Equal(t, example.Expected, example.Executor.PopulateCloneAndWorkingDirEnvironmentVariables(example.Given))
		})
	}
	_ = os.Setenv("TMPDIR", tmpDirToRestore)
}
