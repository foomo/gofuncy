package gofuncy

import (
	"context"
	"sync"
)

// WaitWithReady spawns a goroutine and blocks until fn signals readiness by
// calling ready(), then returns a wait function. If fn returns before calling
// ready(), WaitWithReady unblocks anyway. The wait function blocks until the
// goroutine completes and returns its error. Both the ready and wait functions
// are safe to call multiple times.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.waitwithready".
func WaitWithReady(ctx context.Context, fn func(ctx context.Context, ready ReadyFunc) error, opts ...GoOption) func() error {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.waitwithready"
	}

	ready := make(chan struct{})
	done := make(chan struct{})

	var readyOnce sync.Once

	readyFn := ReadyFunc(func() { readyOnce.Do(func() { close(ready) }) })

	inner := Func(func(ctx context.Context) error {
		return fn(ctx, readyFn)
	})

	run := withContextInjection(inner, o.name)
	run = buildChain(run, &o, "gofuncy.waitwithready", o.callerSkip+3)

	var result error

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			close(ready)
			close(done)

			result = err

			return func() error {
				return result
			}
		}
	}

	go func() {
		defer close(done)

		if o.limiter != nil {
			defer o.limiter.Release(1)
		}

		result = run(ctx)
	}()

	select {
	case <-ready:
	case <-done:
	}

	return func() error {
		<-done
		return result
	}
}
