package gofuncy

import (
	"context"
)

// StartWithStop spawns a fire-and-forget goroutine that receives a stop function.
// Calling stop cancels the goroutine's context, signaling it to shut down.
// The stop function is safe to call multiple times.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.startwithstop".
func StartWithStop(ctx context.Context, fn func(ctx context.Context, stop StopFunc) error, opts ...GoOption) {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.startwithstop"
	}

	if !o.childTrace {
		o.detachedTrace = true
	}

	ctx, cancel := context.WithCancel(ctx)

	inner := Func(func(ctx context.Context) error {
		return fn(ctx, StopFunc(cancel))
	})

	run := withContextInjection(inner, o.name)
	run = buildChain(run, &o, "gofuncy.startwithstop", o.callerSkip+3)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
			cancel()

			return
		}
	}

	go func(ctx context.Context) {
		if o.limiter != nil {
			defer o.limiter.Release(1)
		}

		if ctx.Err() != nil {
			handleError(ctx, ctx.Err(), o.errorHandler, o.l, o.name)
			return
		}

		if err := run(ctx); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
		}
	}(ctx)
}
