package gofuncy

import (
	"context"
)

// GoOptions holds configuration for Go and GoBackground.
type GoOptions struct {
	baseOptions
}

func newGoOptions(opts []Option[GoOptions]) GoOptions {
	o := GoOptions{
		baseOptions: baseOptions{
			name: NameNoName,
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	return o
}

// Go spawns a fire-and-forget goroutine with panic recovery.
// Errors are logged via slog by default; use WithErrorHandler to override.
func Go(ctx context.Context, fn Func, opts ...Option[GoOptions]) {
	o := newGoOptions(opts)

	// build middleware chain (innermost → outermost)
	run := fn

	run = withContextInjection(run, o.name)
	run = withRecover(run)

	for _, m := range o.middlewares {
		run = m(run)
	}

	if o.startedCounter || o.finishedCounter || o.errorCounter || o.activeUpDownCounter || o.durationHistogram {
		m := o.meter()

		if o.startedCounter {
			run = withStartedCounter(run, m, o.name)
		}
		if o.finishedCounter {
			run = withFinishedCounter(run, m, o.name)
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
		run = withTracing(run, &o, o.callerSkip+2)
	}

	if o.timeout > 0 {
		run = withTimeout(run, o.timeout)
	}

	go func(ctx context.Context) {
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
