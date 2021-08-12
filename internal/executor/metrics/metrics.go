// +build !windows !arm

package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"time"
)

var (
	ErrFailedToQueryCPU    = errors.New("failed to query CPU usage")
	ErrFailedToQueryMemory = errors.New("failed to query memory usage")
)

func Run(ctx context.Context) (chan *api.ResourceUtilization, chan error) {
	resultChan := make(chan *api.ResourceUtilization, 1)
	errChan := make(chan error, 1)

	go func() {
		result := &api.ResourceUtilization{}

		pollInterval := 1 * time.Second
		timeSinceStart := time.Duration(0)

		for {
			// CPU usage
			percentages, err := cpu.PercentWithContext(ctx, pollInterval, true)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					resultChan <- result
				} else {
					errChan <- fmt.Errorf("%w: %v", ErrFailedToQueryCPU, err)
				}
				return
			}

			var numCpusUsed float64

			for _, singleCpuUsageInPercents := range percentages {
				numCpusUsed += singleCpuUsageInPercents / 100
			}

			// Memory usage
			virtualMemoryStat, err := mem.VirtualMemoryWithContext(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					resultChan <- result
				} else {
					errChan <- fmt.Errorf("%w: %v", ErrFailedToQueryMemory, err)
				}
				return
			}

			timeSinceStart += pollInterval

			result.CpuChart = append(result.CpuChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            numCpusUsed,
			})
			result.MemoryChart = append(result.MemoryChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            float64(virtualMemoryStat.Used),
			})

			// Gradually increase the poll interval to avoid missing data for
			// short-running tasks, but to preserve memory for long-running tasks
			if timeSinceStart > (5 * time.Minute) {
				pollInterval = 1 * time.Minute
			} else if timeSinceStart > (1 * time.Minute) {
				pollInterval = 10 * time.Second
			}
		}
	}()

	return resultChan, errChan
}
