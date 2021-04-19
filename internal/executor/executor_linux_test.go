// +build !windows !freebsd

package executor

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestPopulateCloneAndWorkingDirEnvironmentVariables(t *testing.T) {
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
