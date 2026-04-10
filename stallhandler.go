package gofuncy

import (
	"context"
	"time"
)

// StallHandler is called when a goroutine exceeds its stall threshold.
// The threshold parameter is the configured stall threshold duration.
type StallHandler func(ctx context.Context, name string, threshold time.Duration)
