package gofuncy

import (
	"context"
	"time"
)

// Start is like [Go] but blocks until the goroutine has actually started
// executing. This is useful in tests where you need the routine to be running
// before proceeding.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.start".
func Start(ctx context.Context, fn Func, opts ...GoOption) {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.start"
	}

	if !o.childTrace {
		o.detachedTrace = true
	}

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.start", o.callerSkip+3)

	recordSpawnEvent(ctx, &o, o.callerSkip+2)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
			return
		}
	}

	started := make(chan struct{})

	go func(ctx context.Context) {
		go func() {
			time.Sleep(time.Millisecond)
			close(started)
		}()

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

	<-started
}
