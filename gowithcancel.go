package gofuncy

import (
	"context"
)

// GoWithCancel spawns a goroutine and returns a stop function. Calling stop cancels
// the goroutine's context, signaling it to shut down. The stop function is
// safe to call multiple times.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.gowithcancel".
func GoWithCancel(ctx context.Context, fn Func, opts ...GoOption) StopFunc {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.gowithcancel"
	}

	if !o.childTrace {
		o.detachedTrace = true
	}

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.gowithcancel", o.callerSkip+3)

	ctx, cancel := context.WithCancel(ctx)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
			cancel()

			return StopFunc(func() {})
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

	return StopFunc(cancel)
}
