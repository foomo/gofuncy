package gofuncy

import (
	"context"
)

// Do executes fn synchronously with the full middleware chain (resilience,
// telemetry, tracing) and returns the error directly. Unlike Go, it does not
// spawn a goroutine.
// The name is used as a metric attribute — use static, low-cardinality values.
func Do(ctx context.Context, name string, fn Func, opts ...GoOption) error {
	o := newGoOptions(opts)
	o.name = name

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.do", o.callerSkip+3)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			return err
		}

		defer o.limiter.Release(1)
	}

	return run(ctx)
}
