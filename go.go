package gofuncy

import (
	"context"
)

// Go spawns a fire-and-forget goroutine with panic recovery.
// Errors are logged via slog by default; use WithErrorHandler to override.
func Go(ctx context.Context, fn Func, opts ...GoOption) {
	o := newGoOptions(opts)

	// build middleware chain (innermost → outermost)
	run := fn

	run = withContextInjection(run, o.name)
	run = withRecover(run)

	for _, m := range o.middlewares {
		run = m(run)
	}

	if o.startedCounter || o.errorCounter || o.activeUpDownCounter || o.durationHistogram {
		m := o.meter()

		if o.startedCounter {
			run = withStartedCounter(run, m, o.name)
		}

		if o.errorCounter {
			run = withErrorCounter(run, m, o.name)
		}

		if o.activeUpDownCounter {
			run = withActiveUpDownCounter(run, m, o.name)
		}

		if o.durationHistogram {
			run = withDurationHistogram(run, m, o.name)
		}
	}

	if o.tracing {
		run = withTracing(run, &o, "gofuncy.go", o.callerSkip+2)
	}

	if o.stallThreshold > 0 {
		run = withStallDetector(run, o.stallThreshold, o.stallHandler, o.meter(), o.l, o.name)
	}

	if o.timeout > 0 {
		run = withTimeout(run, o.timeout)
	}

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
