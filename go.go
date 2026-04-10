package gofuncy

import (
	"context"
)

// Go spawns a fire-and-forget goroutine with panic recovery.
// Errors are logged via slog by default; use WithErrorHandler to override.
// The name is used as a metric attribute — use static, low-cardinality values
// (not request IDs or UUIDs) to avoid unbounded metric series.
func Go(ctx context.Context, name string, fn Func, opts ...GoOption) {
	o := newGoOptions(opts)
	o.name = name

	if !o.childTrace {
		o.detachedTrace = true
	}

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.go", o.callerSkip+3)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
			return
		}
	}

	go func(ctx context.Context) {
		if o.limiter != nil {
			defer o.limiter.Release(1)
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if ctx.Err() != nil {
			handleError(ctx, ctx.Err(), o.errorHandler, o.l, o.name)
			return
		}

		if err := run(ctx); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
		}
	}(ctx)
}
