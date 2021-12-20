package metrics_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second+500*time.Millisecond)
	defer cancel()

	resultChan := metrics.Run(ctx, nil)

	result := <-resultChan

	for i, err := range result.Errors {
		fmt.Printf("Error #%d: %v\n", i, err)
	}
	require.Empty(t, result.Errors, "we should never get errors here, but got %d", len(result.Errors))
	require.Len(t, result.ResourceUtilization.CpuChart, 4)
	require.Len(t, result.ResourceUtilization.MemoryChart, 4)
}
