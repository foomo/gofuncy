package gofuncy

import (
	"context"
)

// WaitWithStop spawns a goroutine that receives a stop function and returns a
// wait function. Calling stop cancels the goroutine's context. The wait
// function blocks until the goroutine completes and returns its error. Both
// functions are safe to call multiple times.
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.waitwithstop".
func WaitWithStop(ctx context.Context, fn func(ctx context.Context, stop StopFunc) error, opts ...GoOption) func() error {
	o := newGoOptions(opts)
	if o.name == "" {
		o.name = "gofuncy.waitwithstop"
	}

	ctx, cancel := context.WithCancel(ctx)

	inner := Func(func(ctx context.Context) error {
		return fn(ctx, StopFunc(cancel))
	})

	run := withContextInjection(inner, o.name)
	run = buildChain(run, &o, "gofuncy.waitwithstop", o.callerSkip+3)

	var (
		result error
		done   = make(chan struct{})
	)

	if o.limiter != nil {
		if err := o.limiter.Acquire(ctx, 1); err != nil {
			close(done)
			cancel()

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
