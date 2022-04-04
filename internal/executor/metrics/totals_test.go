package metrics_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTotals(t *testing.T) {
	ctx := context.Background()

	expectedNumCpusTotal, err := cpu.Counts(true)
	require.NoError(t, err)
	expectedAmountMemory, err := mem.VirtualMemory()
	require.NoError(t, err)

	numCpusTotal, amountMemoryTotal, err := metrics.Totals(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, expectedNumCpusTotal, numCpusTotal)
	assert.EqualValues(t, expectedAmountMemory.Total, amountMemoryTotal)
}
