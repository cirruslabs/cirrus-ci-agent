package executor

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateRelativeToAllInWorkingDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	scopedToWorkingDir, err := isScopedToWorkingDir(dir, []ProcessedArtifactPath{
		{
			FinalPaths: []string{
				filepath.Join(dir, "some-intermediate-dir", "artifact.txt"),
			},
		},
	})
	require.NoError(t, err)
	require.True(t, scopedToWorkingDir)
}

func TestCalculateRelativeToSomeInWorkingDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	scopedToWorkingDir, err := isScopedToWorkingDir(dir, []ProcessedArtifactPath{
		{
			FinalPaths: []string{
				filepath.Join(dir, "some-intermediate-dir", "artifact.txt"),
			},
		},
		{
			FinalPaths: []string{
				string(filepath.Separator),
			},
		},
	})
	require.NoError(t, err)
	require.False(t, scopedToWorkingDir)
}
