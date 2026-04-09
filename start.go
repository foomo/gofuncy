package gofuncy

import (
	"context"
)

// Wait spawns a goroutine with the full middleware chain and returns a wait
// function. Calling the wait function blocks until the goroutine completes and
// returns its error. The wait function is safe to call multiple times and from
// multiple goroutines — it always returns the same result.
// The name is used as a metric attribute — use static, low-cardinality values.
func Wait(ctx context.Context, name string, fn Func, opts ...GoOption) func() error {
	o := newGoOptions(opts)
	o.name = name

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.wait", o.callerSkip+3)

	var (
		result error
		done   = make(chan struct{})
	)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
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

	return func() error {
		<-done
		return result
	}
}
