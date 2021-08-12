package metrics_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultChan, errChan := metrics.Run(ctx)

	select {
	case result := <-resultChan:
		require.Len(t, result.CpuChart, 4)
		require.Len(t, result.MemoryChart, 4)
	case err := <-errChan:
		require.Fail(t, "we should never get an error here, but got %v", err)
	}
}
