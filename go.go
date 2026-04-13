package gofuncy

import (
	"context"
)

// Go spawns a fire-and-forget goroutine with panic recovery.
// Errors are logged via slog by default; use WithErrorHandler to override.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.go".
func Go(ctx context.Context, fn Func, opts ...GoOption) {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.go"
	}

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
