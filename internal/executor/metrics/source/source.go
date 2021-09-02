package source

import (
	"context"
	"time"
)

type CPU interface {
	NumCpusUsed(ctx context.Context, pollInterval time.Duration) (float64, error)
}

type Memory interface {
	AmountMemoryUsed(ctx context.Context) (float64, error)
}
