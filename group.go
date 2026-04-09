package gofuncy

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
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

	wg   sync.WaitGroup
	sem  chan struct{}
	once sync.Once

	mu   sync.Mutex
	errs []error
	err  error

	span  trace.Span
	start time.Time
}

// NewGroup creates a new Group with the given context and options.
// The name is used as a metric attribute — use static, low-cardinality values.
func NewGroup(ctx context.Context, name string, opts ...GroupOption) *Group {
	o := newGroupOptions(opts)
	o.name = name

	g := &Group{
		ctx: ctx,
		o:   o,
	}

	if o.failFast {
		g.ctx, g.cancel = context.WithCancel(ctx) //nolint:gosec // cancel is called in Wait()
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

	run := withContextInjection(fn, o.name)
	run = buildChain(run, &o, "gofuncy.group.add", 3)

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
		select {
		case g.sem <- struct{}{}:
		case <-g.ctx.Done():
			g.mu.Lock()
			g.errs[idx] = g.ctx.Err()
			g.mu.Unlock()

			return
		}
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
// It is safe to call multiple times — the result is computed once.
func (g *Group) Wait() error {
	g.once.Do(func() {
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

			groupDuration, err := gofuncyconv.NewGroupsDuration(g.o.meter())
			if err != nil {
				otel.Handle(err)
			}

			groupDuration.Record(context.WithoutCancel(g.ctx), dur, g.o.name, hasErr)
		}

		if g.cancel != nil {
			g.cancel()
		}

		g.err = errors.Join(g.errs...)
	})

	return g.err
}
