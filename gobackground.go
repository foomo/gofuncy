package gofuncy

import (
	"context"
)

// GoBackground is like Go but detaches from the parent context's cancellation.
// The goroutine will continue running even if the parent context is canceled.
func GoBackground(ctx context.Context, fn Func, opts ...*OptionsBuilder) {
	Go(context.WithoutCancel(ctx), fn, opts...)
}
