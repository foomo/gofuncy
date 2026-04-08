package gofuncy

import (
	"context"
	"time"
)

// StallHandler is called when a goroutine exceeds its stall threshold.
type StallHandler func(ctx context.Context, name string, elapsed time.Duration)
