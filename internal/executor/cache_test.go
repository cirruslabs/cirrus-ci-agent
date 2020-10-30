package executor_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeduplicatePaths(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          []string
		ExpectedOutput []string
	}{
		{
			Name: "simple",
			Input: []string{
				"/tmp/node_modules/module/node_modules",
				"/tmp/node_modules",
			},
			ExpectedOutput: []string{
				"/tmp/node_modules",
			},
		},
		{
			Name: "path-aware comparison",
			Input: []string{
				"/tmp/node_modules/module/node_modules",
				"/tmp/node_mod",
			},
			ExpectedOutput: []string{
				"/tmp/node_mod",
				"/tmp/node_modules/module/node_modules",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			assert.Equal(t, testCase.ExpectedOutput, executor.DeduplicatePaths(testCase.Input))
		})
	}
}
