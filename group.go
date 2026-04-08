package gofuncy

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

// Group manages a set of concurrently executing functions with shared
// lifecycle control.
type Group struct {
	ctx    context.Context //nolint:containedctx
	cancel context.CancelFunc
	o      options

	wg  sync.WaitGroup
	sem chan struct{}

	mu   sync.Mutex
	errs []error

	span  trace.Span
	start time.Time
}

// NewGroup creates a new Group with the given context and options.
func NewGroup(ctx context.Context, name string, opts ...GroupOption) *Group {
	o := newGroupOptions(opts)
	o.name = name

	g := &Group{
		ctx: ctx,
		o:   o,
	}

	if o.failFast {
		g.ctx, g.cancel = context.WithCancel(ctx)
	}

	if o.limit > 0 {
		g.sem = make(chan struct{}, o.limit)
	}

	if o.tracing || o.durationHistogram {
		g.start = time.Now()
	}

	if o.tracing {
		g.ctx, g.span = o.tracer().Start(g.ctx, "gofuncy.group "+o.name) //nolint:spancheck
	}

	return g
}

// Add spawns a goroutine to execute fn immediately.
// Per-function opts are merged on top of the group options (additive).
// User middlewares and panic recovery are applied per fn.
func (g *Group) Add(name string, fn Func, opts ...GoOption) {
	o := g.o
	if len(opts) > 0 {
		o = o.merge(newGoOverrideOptions(opts))
	}
	o.name = name

	// build middleware chain (innermost → outermost)
	run := fn
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
		run = withTracing(run, &o, "gofuncy.group.add", 2)
	}

	if o.stallThreshold > 0 {
		run = withStallDetector(run, o.stallThreshold, o.stallHandler, o.meter(), o.l, o.name)
	}

	if o.timeout > 0 {
		run = withTimeout(run, o.timeout)
	}

	g.mu.Lock()
	idx := len(g.errs)
	g.errs = append(g.errs, nil)
	g.mu.Unlock()

	if o.limiter != nil {
		if err := o.limiter.Acquire(g.ctx, 1); err != nil {
			g.mu.Lock()
			g.errs[idx] = err
			g.mu.Unlock()

			if g.cancel != nil {
				g.cancel()
			}

			return
		}
	} else if g.sem != nil {
		g.sem <- struct{}{}
	}

	g.wg.Go(func() {
		if o.limiter != nil {
			defer o.limiter.Release(1)
		} else if g.sem != nil {
			defer func() { <-g.sem }()
		}

		err := run(g.ctx)

		g.mu.Lock()
		g.errs[idx] = err
		g.mu.Unlock()

		if err != nil && g.cancel != nil {
			g.cancel()
		}
	})
}

// Wait blocks until all added functions complete and returns the joined errors.
func (g *Group) Wait() error {
	g.wg.Wait()

	hasErr := false

	if g.span != nil {
		g.span.SetAttributes(semconv.GroupSize(len(g.errs)))

		for _, e := range g.errs {
			if e != nil {
				hasErr = true

				g.span.RecordError(e)
			}
		}

		if hasErr {
			g.span.SetStatus(codes.Error, "group completed with errors")
		}

		g.span.End()
	}

	if g.o.durationHistogram {
		if !hasErr {
			for _, e := range g.errs {
				if e != nil {
					hasErr = true

					break
				}
			}
		}

		dur := time.Since(g.start).Truncate(time.Millisecond).Seconds()

		groupDuration, _ := gofuncyconv.NewGroupsDuration(g.o.meter())
		groupDuration.Record(context.WithoutCancel(g.ctx), dur, g.o.name, hasErr, len(g.errs))
	}

	if g.cancel != nil {
		g.cancel()
	}

	return errors.Join(g.errs...)
}
