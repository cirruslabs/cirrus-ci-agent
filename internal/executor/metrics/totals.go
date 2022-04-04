package metrics

import (
	"context"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func Totals(ctx context.Context) (uint64, uint64, error) {
	perCpuStat, err := cpu.TimesWithContext(ctx, true)
	if err != nil {
		return 0, 0, err
	}

	virtualMemoryStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	return uint64(len(perCpuStat)), virtualMemoryStat.Total, nil
}
