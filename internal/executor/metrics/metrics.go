//go:build !windows || (!arm && !arm64)
// +build !windows !arm,!arm64

package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/cgroup/cpu"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/cgroup/memory"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/cgroup/resolver"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/metrics/source/system"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"log"
	"runtime"
	"time"
)

var (
	ErrFailedToQueryCPU    = errors.New("failed to query CPU usage")
	ErrFailedToQueryMemory = errors.New("failed to query memory usage")
)

type Result struct {
	Errors              []error
	ResourceUtilization api.ResourceUtilization
}

func Run(ctx context.Context, logger logrus.FieldLogger) chan *Result {
	resultChan := make(chan *Result, 1)

	var cpuSource source.CPU
	var memorySource source.Memory

	systemSource := system.New()
	cpuSource = systemSource
	memorySource = systemSource

	resolver, err := resolver.New()
	if err != nil {
		if runtime.GOOS == "linux" {
			log.Printf("cgroup resolver initialization failed (%v), falling back to system-wide metrics collection",
				err)
		}
	} else {
		cpuSrc, err := cpu.NewCPU(resolver)
		if err == nil {
			if logger != nil {
				logger.Infof("CPU metrics are now cgroup-aware")
			}
			cpuSource = cpuSrc
		}

		memorySrc, err := memory.NewMemory(resolver)
		if err == nil {
			if logger != nil {
				logger.Infof("memory metrics are now cgroup-aware")
			}
			memorySource = memorySrc
		}
	}

	go func() {
		result := &Result{}

		pollInterval := 1 * time.Second
		startTime := time.Now()

		for {
			// CPU usage
			numCpusUsed, err := cpuSource.NumCpusUsed(ctx, pollInterval)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					resultChan <- result
					return
				}

				numCpusUsed = -1.0
				result.Errors = append(result.Errors,
					fmt.Errorf("%w using %s: %v", ErrFailedToQueryCPU, cpuSource.Name(), err))
			}

			// Memory usage
			amountMemoryUsed, err := memorySource.AmountMemoryUsed(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					resultChan <- result
				}

				amountMemoryUsed = -1.0
				result.Errors = append(result.Errors,
					fmt.Errorf("%w using %s: %v", ErrFailedToQueryMemory, memorySource.Name(), err))
			}

			if logger != nil {
				logger.Infof("CPUs used: %.2f, CPU usage: %.2f%%, memory used: %s", numCpusUsed, numCpusUsed*100.0,
					humanize.Bytes(uint64(amountMemoryUsed)))
			}

			timeSinceStart := time.Since(startTime)

			result.ResourceUtilization.CpuChart = append(result.ResourceUtilization.CpuChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            numCpusUsed,
			})
			result.ResourceUtilization.MemoryChart = append(result.ResourceUtilization.MemoryChart, &api.ChartPoint{
				SecondsFromStart: uint32(timeSinceStart.Seconds()),
				Value:            amountMemoryUsed,
			})

			// Gradually increase the poll interval to avoid missing data for
			// short-running tasks, but to preserve memory for long-running tasks
			if timeSinceStart > (1 * time.Minute) {
				pollInterval = 10 * time.Second
			}
		}
	}()

	return resultChan
}
