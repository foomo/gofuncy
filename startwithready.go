package gofuncy

import (
	"context"
	"sync"
	"time"
)

// StartWithReady spawns a goroutine and blocks until fn signals readiness by calling
// ready(). If fn returns before calling ready(), StartWithReady unblocks anyway.
// The ready function is safe to call multiple times.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.startwithready".
func StartWithReady(ctx context.Context, fn func(ctx context.Context, ready ReadyFunc) error, opts ...GoOption) {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.startwithready"
	}

	if !o.childTrace {
		o.detachedTrace = true
	}

	ready := make(chan struct{})
	done := make(chan struct{})

	var readyOnce sync.Once

	readyFn := func() {
		readyOnce.Do(func() {
			time.Sleep(time.Microsecond)
			close(ready)
		})
	}

	inner := Func(func(ctx context.Context) error {
		return fn(ctx, readyFn)
	})

	run := withContextInjection(inner, o.name)
	run = buildChain(run, &o, "gofuncy.startwithready", o.callerSkip+3)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			handleError(ctx, err, o.errorHandler, o.l, o.name)
			return
		}
	}

	go func(ctx context.Context) {
		defer close(done)

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

	select {
	case <-ready:
	case <-done:
	}
}
