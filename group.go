package gofuncy

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

// Group runs all functions concurrently and waits for all to complete.
// Returns errors.Join of all failures, nil if all succeed.
// Use GroupOption().WithFailFast() to cancel remaining on first error.
// Use GroupOption().WithLimit(n) to bound concurrent goroutines.
func Group(ctx context.Context, fns []Func, opts ...Option[GroupOptions]) error {
	if len(fns) == 0 {
		return nil
	}

	o := newGroupOptions(opts)

	errs := run(ctx, fns, &o.concurrentOptions)

	return errors.Join(errs...)
}

// GroupBackground is like Group but detaches from the parent context's cancellation.
// The goroutines will continue running even if the parent context is canceled.
func GroupBackground(ctx context.Context, fns []Func, opts ...Option[GroupOptions]) error {
	return Group(context.WithoutCancel(ctx), fns, opts...)
}

// run is the shared execution engine for Group and Map.
func run(ctx context.Context, fns []Func, o *concurrentOptions) []error {
	errs := make([]error, len(fns))

	if o.failFast {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		runAll(ctx, fns, errs, o, cancel)
	} else {
		runAll(ctx, fns, errs, o, nil)
	}

	return errs
}

func runAll(ctx context.Context, fns []Func, errs []error, o *concurrentOptions, cancelOnErr context.CancelFunc) {
	var wg sync.WaitGroup
	wg.Add(len(fns))

	// optional semaphore for concurrency limiting
	var sem chan struct{}
	if o.limit > 0 {
		sem = make(chan struct{}, o.limit)
	}

	// telemetry
	var (
		span  trace.Span
		start time.Time
	)

	if o.tracing || o.durationMetric {
		start = time.Now()
	}

	if o.tracing {
		var traceCtx context.Context

		traceCtx, span = resolveTracer(o.tracerProvider).Start(ctx, "gofuncy.group "+o.name,
			trace.WithAttributes(
				semconv.GroupSize(len(fns)),
			),
		)
		ctx = traceCtx

		defer func() {
			hasErr := false

			for _, e := range errs {
				if e != nil {
					hasErr = true

					span.RecordError(e)

					break
				}
			}

			if hasErr {
				span.SetStatus(codes.Error, "group completed with errors")
			}

			span.End()
		}()
	}

	for i, fn := range fns {
		if sem != nil {
			sem <- struct{}{}
		}

		go func(i int, fn Func) {
			defer wg.Done()

			if sem != nil {
				defer func() { <-sem }()
			}

			var err error

			defer func() { errs[i] = err }()

			defer recoverError(&err)

			err = fn(ctx)

			if err != nil && cancelOnErr != nil {
				cancelOnErr()
			}
		}(i, fn)
	}

	wg.Wait()

	// record group duration metric
	if o.durationMetric {
		inst := resolveInstrumentation(o.meterProvider)

		hasErr := false

		for _, e := range errs {
			if e != nil {
				hasErr = true

				break
			}
		}

		inst.recordGroupDuration(context.WithoutCancel(ctx),
			time.Since(start).Truncate(time.Millisecond).Seconds(),
			o.name,
			hasErr,
			len(fns),
		)
	}
}
