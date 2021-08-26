package system

import (
	"context"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"time"
)

type System struct{}

func New() *System {
	return &System{}
}

func (system *System) NumCpusUsed(ctx context.Context, pollInterval time.Duration) (float64, error) {
	percentages, err := cpu.PercentWithContext(ctx, pollInterval, true)
	if err != nil {
		return 0, err
	}

	var numCpusUsed float64

	for _, singleCpuUsageInPercents := range percentages {
		numCpusUsed += singleCpuUsageInPercents / 100
	}

	return numCpusUsed, nil
}

func (system *System) AmountMemoryUsed(ctx context.Context) (float64, error) {
	virtualMemoryStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, err
	}

	return float64(virtualMemoryStat.Used), nil
}
