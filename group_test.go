package gofuncy_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slogx "github.com/foomo/go/slog"
	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossymetric"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/semaphore"
)

func ExampleNewGroup() {
	g := gofuncy.NewGroup(context.Background())

	g.Add(func(ctx context.Context) error {
		fmt.Println("a")
		return nil
	})
	g.Add(func(ctx context.Context) error {
		fmt.Println("b")
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Println("error:", err)
	}

	// Unordered output:
	// a
	// b
}

func TestGroup_basic(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	g := gofuncy.NewGroup(t.Context())

	for range 3 {
		g.Add(func(ctx context.Context) error {
			count.Add(1)
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(3), count.Load())
}

func TestGroup_empty(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_errors(t *testing.T) {
	t.Parallel()

	errA := errors.New("error a")
	errB := errors.New("error b")

	g := gofuncy.NewGroup(t.Context())

	g.Add(func(ctx context.Context) error {
		return errA
	})
	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return errB
	})

	err := g.Wait()
	require.Error(t, err)
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
}

func TestGroup_failFast(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithFailFast(),
	)

	g.Add(func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("first error")
	})
	g.Add(func(ctx context.Context) error {
		<-started
		<-ctx.Done()

		return ctx.Err()
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGroup_withLimit(t *testing.T) {
	t.Parallel()

	const limit = 2

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
	)

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithLimit(limit),
	)

	for range 10 {
		g.Add(func(ctx context.Context) error {
			cur := active.Add(1)

			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			active.Add(-1)

			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))
}

func TestGroup_panicRecovery(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())

	g.Add(func(ctx context.Context) error {
		panic("group panic")
	})

	err := g.Wait()
	require.Error(t, err)

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "group panic", panicErr.Value)
}

func TestGroup_canceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	g := gofuncy.NewGroup(ctx)

	g.Add(func(ctx context.Context) error {
		return ctx.Err()
	})

	err := g.Wait()
	require.ErrorIs(t, err, context.Canceled)
}

func TestGroup_withMiddleware(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	mw := func(fn gofuncy.Func) gofuncy.Func {
		return func(ctx context.Context) error {
			calls.Add(1)
			return fn(ctx)
		}
	}

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithMiddleware(mw),
	)

	for range 3 {
		g.Add(func(ctx context.Context) error {
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestGroup_withName(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("test-group"),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_addWithMiddleware(t *testing.T) {
	t.Parallel()

	var (
		groupCalls atomic.Int32
		addCalls   atomic.Int32
	)

	groupMW := func(fn gofuncy.Func) gofuncy.Func {
		return func(ctx context.Context) error {
			groupCalls.Add(1)
			return fn(ctx)
		}
	}

	addMW := func(fn gofuncy.Func) gofuncy.Func {
		return func(ctx context.Context) error {
			addCalls.Add(1)
			return fn(ctx)
		}
	}

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithMiddleware(groupMW),
	)

	// fn without per-function middleware
	g.Add(func(ctx context.Context) error {
		return nil
	})

	// fn with per-function middleware
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithMiddleware(addMW))

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(2), groupCalls.Load(), "group middleware should run for both functions")
	assert.Equal(t, int32(1), addCalls.Load(), "add middleware should run only for the opted-in function")
}

func TestGroup_addWithName(t *testing.T) {
	t.Parallel()

	var (
		names []string
		mu    sync.Mutex
	)

	mw := func(fn gofuncy.Func) gofuncy.Func {
		return func(ctx context.Context) error {
			mu.Lock()

			names = append(names, gofuncy.NameFromContext(ctx))
			mu.Unlock()

			return fn(ctx)
		}
	}

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("group"),
		gofuncy.WithMiddleware(mw),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithName("task-b"))

	err := g.Wait()
	require.NoError(t, err)

	// Both functions should have received the group name since
	// context injection is only used in Go(), not Group.Add()
	assert.Len(t, names, 2)
}

// ------------------------------------------------------------------------------------------------
// ~ Tracing
// ------------------------------------------------------------------------------------------------

func TestGroup_withTracing(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t, glossytrace.WithSpanAttributes()))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("traced-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add(func(ctx context.Context) error {
		time.Sleep(time.Millisecond)

		sp := trace.SpanFromContext(ctx)
		sp.AddEvent("test event")

		return nil
	}, gofuncy.WithName("traced-task"))
	g.Add(func(ctx context.Context) error {
		time.Sleep(time.Millisecond)

		sp := trace.SpanFromContext(ctx)
		sp.AddEvent("test event")

		return nil
	}, gofuncy.WithName("traced-task"))

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_withTracingErrors(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("traced-group-errors"),
		gofuncy.WithLogger(l),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("traced error")
	})

	err := g.Wait()
	require.Error(t, err)
}

// ------------------------------------------------------------------------------------------------
// ~ Metrics
// ------------------------------------------------------------------------------------------------

func TestGroup_withStartedCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("started-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithStartedCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 3 {
		g.Add(func(ctx context.Context) error {
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_withFinishedCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("finished-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 3 {
		g.Add(func(ctx context.Context) error {
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_withErrorCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("error-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithErrorCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("counter error")
	})

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_withActiveUpDownCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("active-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 3 {
		g.Add(func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_withDurationHistogram(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("duration-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 3 {
		g.Add(func(ctx context.Context) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_withDurationHistogramErrors(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("duration-group-errors"),
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("duration error")
	})

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_withAllMetrics(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("all-metrics-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithStartedCounter(),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithErrorCounter(),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("all metrics error")
	})

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_withTracingAndMetrics(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("full-telemetry-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithStartedCounter(),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithErrorCounter(),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("telemetry error")
	})

	err := g.Wait()
	require.Error(t, err)
}

// ------------------------------------------------------------------------------------------------
// ~ Per-function telemetry overrides
// ------------------------------------------------------------------------------------------------

func TestGroup_addWithStartedCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("add-started-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithMeterProvider(mp),
	)

	// only this function enables the started counter
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithStartedCounter())

	g.Add(func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_addWithAllMetrics(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("add-all-metrics-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithMeterProvider(mp),
	)

	// per-function: enable all metrics with a custom name
	g.Add(func(ctx context.Context) error {
		return nil
	},
		gofuncy.WithName("task-a"),
		gofuncy.WithStartedCounter(),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithErrorCounter(),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithDurationHistogram(),
	)

	// no per-function options
	g.Add(func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_addWithLogger(t *testing.T) {
	t.Parallel()

	groupLogger := slog.New(slogx.NewTestHandler(t))
	addLogger := slog.New(slogx.NewTestHandler(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("add-logger-group"),
		gofuncy.WithLogger(groupLogger),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	})
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithLogger(addLogger))

	err := g.Wait()
	require.NoError(t, err)
}

// ------------------------------------------------------------------------------------------------
// ~ Fail-fast with telemetry
// ------------------------------------------------------------------------------------------------

func TestGroup_failFastWithTracing(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	started := make(chan struct{})

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("failfast-traced"),
		gofuncy.WithLogger(l),
		gofuncy.WithFailFast(),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add(func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("fail fast error")
	})
	g.Add(func(ctx context.Context) error {
		<-started
		<-ctx.Done()

		return ctx.Err()
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGroup_failFastWithMetrics(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	started := make(chan struct{})

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("failfast-metrics"),
		gofuncy.WithLogger(l),
		gofuncy.WithFailFast(),
		gofuncy.WithStartedCounter(),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithErrorCounter(),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add(func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("fail fast metric error")
	})
	g.Add(func(ctx context.Context) error {
		<-started
		<-ctx.Done()

		return ctx.Err()
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ------------------------------------------------------------------------------------------------
// ~ Limit with telemetry
// ------------------------------------------------------------------------------------------------

func TestGroup_withLimitAndMetrics(t *testing.T) {
	t.Parallel()

	const limit = 2

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
	)

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("limit-metrics"),
		gofuncy.WithLogger(l),
		gofuncy.WithLimit(limit),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithStartedCounter(),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 6 {
		g.Add(func(ctx context.Context) error {
			cur := active.Add(1)

			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			active.Add(-1)

			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))
}

// ------------------------------------------------------------------------------------------------
// ~ Per-function tracing and timeout
// ------------------------------------------------------------------------------------------------

func TestGroup_addWithTimeout(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())

	g.Add(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(50*time.Millisecond))

	g.Add(func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGroup_addWithTracing(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	g := gofuncy.NewGroup(t.Context(),
		gofuncy.WithName("traced-group"),
		gofuncy.WithLogger(l),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithTracing(),
		gofuncy.WithName("task-a"),
		gofuncy.WithTracerProvider(tp),
	)
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithTracing(),
		gofuncy.WithName("task-b"),
		gofuncy.WithTracerProvider(tp),
	)

	err := g.Wait()
	require.NoError(t, err)
}

// ------------------------------------------------------------------------------------------------
// ~ Shared limiter
// ------------------------------------------------------------------------------------------------

func TestGroup_withLimiter(t *testing.T) {
	t.Parallel()

	const limit = 2

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
	)

	sem := semaphore.NewWeighted(int64(limit))

	// Two independent groups sharing one limiter.
	g1 := gofuncy.NewGroup(t.Context(),
		gofuncy.WithLimiter(sem),
	)
	g2 := gofuncy.NewGroup(t.Context(),
		gofuncy.WithLimiter(sem),
	)

	work := func(ctx context.Context) error {
		cur := active.Add(1)

		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		active.Add(-1)

		return nil
	}

	for range 5 {
		g1.Add(work)
	}

	for range 5 {
		g2.Add(work)
	}

	err := g1.Wait()
	require.NoError(t, err)

	err = g2.Wait()
	require.NoError(t, err)

	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))
}

func TestGroup_withLimiterCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	sem := semaphore.NewWeighted(1)

	g := gofuncy.NewGroup(ctx,
		gofuncy.WithLimiter(sem),
	)

	started := make(chan struct{})

	// First task holds the semaphore.
	g.Add(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()

		return ctx.Err()
	})

	// Wait for first task to start, then cancel the context
	// so the second Add's Acquire unblocks with an error.
	<-started
	cancel()

	g.Add(func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
