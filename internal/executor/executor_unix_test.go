// +build !windows !freebsd

package executor_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetExpandedScriptEnvironment(t *testing.T) {
	e := &executor.Executor{}
	examples := []struct {
		Description     string
		Given, Expected map[string]string
	}{
		{
			"empty",
			map[string]string{},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/cirrus-ci-build",
				"CIRRUS_WORKING_DIR": "/tmp/cirrus-ci-build",
			},
		},
		{
			"only working",
			map[string]string{
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
		},
		{
			"only working (monorepo)",
			map[string]string{
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/cirrus-ci-build",
				"CIRRUS_WORKING_DIR": "/tmp/cirrus-ci-build/foo",
			},
		},
		{
			"only clone",
			map[string]string{
				"CIRRUS_CLONE_DIR": "/tmp/foo",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo",
			},
		},
		{
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
			"both (monorepo)",
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "$CIRRUS_CLONE_DIR/bar",
			},
			map[string]string{
				"CIRRUS_CLONE_DIR":   "/tmp/foo",
				"CIRRUS_WORKING_DIR": "/tmp/foo/bar",
			},
		},
	}

	for _, example := range examples {
		t.Run(example.Description, func(t *testing.T) {
			require.Equal(t, example.Expected, e.GetExpandedScriptEnvironment(example.Given))
		})
	}
}
