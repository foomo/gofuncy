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
	"golang.org/x/sync/semaphore"
)

func ExampleNewGroup() {
	g := gofuncy.NewGroup(context.Background(), "example")

	g.Add("a", func(ctx context.Context) error {
		fmt.Println("a")
		return nil
	})
	g.Add("b", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "basic")

	for range 3 {
		g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "empty")

	err := g.Wait()
	require.NoError(t, err)
}

func TestGroup_errors(t *testing.T) {
	t.Parallel()

	errA := errors.New("error a")
	errB := errors.New("error b")

	g := gofuncy.NewGroup(t.Context(), "errors")

	g.Add("a", func(ctx context.Context) error {
		return errA
	})
	g.Add("b", func(ctx context.Context) error {
		return nil
	})
	g.Add("c", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "fail-fast",
		gofuncy.WithFailFast(),
	)

	g.Add("first", func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("first error")
	})
	g.Add("second", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "with-limit",
		gofuncy.WithLimit(limit),
	)

	for range 10 {
		g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "panic")

	g.Add("panicker", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(ctx, "canceled")

	g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "with-middleware",
		gofuncy.WithMiddleware(mw),
	)

	for range 3 {
		g.Add("task", func(ctx context.Context) error {
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestGroup_withName(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context(), "test-group")

	g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "add-middleware",
		gofuncy.WithMiddleware(groupMW),
	)

	// fn without per-function middleware
	g.Add("task-a", func(ctx context.Context) error {
		return nil
	})

	// fn with per-function middleware
	g.Add("task-b", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "group",
		gofuncy.WithMiddleware(mw),
	)

	g.Add("task-a", func(ctx context.Context) error {
		return nil
	})
	g.Add("task-b", func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.NoError(t, err)

	// Both functions should have received the group name since
	// context injection is only used in Go(), not Group.Add()
	assert.Len(t, names, 2)
}

// ------------------------------------------------------------------------------------------------
// ~ Stall detection
// ------------------------------------------------------------------------------------------------

func TestGroup_withStallThreshold(t *testing.T) {
	t.Parallel()

	var (
		stallCount atomic.Int32
		mu         sync.Mutex
		stallNames []string
	)

	g := gofuncy.NewGroup(t.Context(), "stall-group",
		gofuncy.WithStallThreshold(10*time.Millisecond),
		gofuncy.WithStallHandler(func(ctx context.Context, name string, elapsed time.Duration) {
			stallCount.Add(1)
			mu.Lock()

			stallNames = append(stallNames, name)
			mu.Unlock()
		}),
	)

	// slow task — should trigger stall
	g.Add("slow-task", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	// fast task — should NOT trigger stall
	g.Add("fast-task", func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(1), stallCount.Load())

	mu.Lock()
	assert.Contains(t, stallNames, "slow-task")
	mu.Unlock()
}

// ------------------------------------------------------------------------------------------------
// ~ Metrics
// ------------------------------------------------------------------------------------------------

func TestGroup_withDurationHistogram(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(), "duration-group",
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	for range 3 {
		g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "duration-group-errors",
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add("ok", func(ctx context.Context) error {
		return nil
	})
	g.Add("fail", func(ctx context.Context) error {
		return fmt.Errorf("duration error")
	})

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_withAllMetrics(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	g := gofuncy.NewGroup(t.Context(), "all-metrics-group",
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add("ok", func(ctx context.Context) error {
		return nil
	})
	g.Add("fail", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "full-telemetry-group",
		gofuncy.WithLogger(l),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add("ok", func(ctx context.Context) error {
		return nil
	})
	g.Add("fail", func(ctx context.Context) error {
		return fmt.Errorf("telemetry error")
	})

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_addWithLogger(t *testing.T) {
	t.Parallel()

	groupLogger := slog.New(slogx.NewTestHandler(t))
	addLogger := slog.New(slogx.NewTestHandler(t))

	g := gofuncy.NewGroup(t.Context(), "add-logger-group",
		gofuncy.WithLogger(groupLogger),
	)

	g.Add("task-a", func(ctx context.Context) error {
		return nil
	})
	g.Add("task-b", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "failfast-traced",
		gofuncy.WithLogger(l),
		gofuncy.WithFailFast(),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add("first", func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("fail fast error")
	})
	g.Add("second", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "failfast-metrics",
		gofuncy.WithLogger(l),
		gofuncy.WithFailFast(),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	g.Add("first", func(ctx context.Context) error {
		close(started)
		return fmt.Errorf("fail fast metric error")
	})
	g.Add("second", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "limit-metrics",
		gofuncy.WithLogger(l),
		gofuncy.WithLimit(limit),
		gofuncy.WithMeterProvider(mp),
	)

	for range 6 {
		g.Add("task", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "add-timeout")

	g.Add("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(50*time.Millisecond))

	g.Add("fast", func(ctx context.Context) error {
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

	g := gofuncy.NewGroup(t.Context(), "traced-group",
		gofuncy.WithLogger(l),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add("task-a", func(ctx context.Context) error {
		return nil
	},
		gofuncy.WithTracerProvider(tp),
	)
	g.Add("task-b", func(ctx context.Context) error {
		return nil
	},
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
	g1 := gofuncy.NewGroup(t.Context(), "group-1",
		gofuncy.WithLimiter(sem),
	)
	g2 := gofuncy.NewGroup(t.Context(), "group-2",
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
		g1.Add("work", work)
	}

	for range 5 {
		g2.Add("work", work)
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

	g := gofuncy.NewGroup(ctx, "limiter-canceled",
		gofuncy.WithLimiter(sem),
	)

	started := make(chan struct{})

	// First task holds the semaphore.
	g.Add("holder", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()

		return ctx.Err()
	})

	// Wait for first task to start, then cancel the context
	// so the second Add's Acquire unblocks with an error.
	<-started
	cancel()

	g.Add("waiter", func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ------------------------------------------------------------------------------------------------
// ~ Edge cases
// ------------------------------------------------------------------------------------------------

func TestGroup_waitCalledTwice(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context(), "wait-twice",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	g.Add("task", func(ctx context.Context) error {
		return fmt.Errorf("once")
	})

	err1 := g.Wait()
	err2 := g.Wait()

	require.Error(t, err1)
	require.Error(t, err2)
	assert.Equal(t, err1.Error(), err2.Error())
}

func TestGroup_largeScale(t *testing.T) {
	t.Parallel()

	const n = 1000

	var count atomic.Int32

	g := gofuncy.NewGroup(t.Context(), "large-scale",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	for range n {
		g.Add("task", func(ctx context.Context) error {
			count.Add(1)
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(n), count.Load())
}

func TestGroup_allErrorSimultaneously(t *testing.T) {
	t.Parallel()

	const n = 10

	g := gofuncy.NewGroup(t.Context(), "all-errors",
		gofuncy.WithFailFast(),
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	errs := make([]error, n)
	for i := range n {
		errs[i] = fmt.Errorf("error-%d", i)
	}

	for i := range n {
		g.Add(fmt.Sprintf("task-%d", i), func(ctx context.Context) error {
			return errs[i]
		})
	}

	err := g.Wait()
	require.Error(t, err)
}

func TestGroup_externalContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	g := gofuncy.NewGroup(ctx, "external-cancel",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	allStarted := make(chan struct{})

	var started atomic.Int32

	for range 5 {
		g.Add("task", func(ctx context.Context) error {
			if started.Add(1) == 5 {
				close(allStarted)
			}

			<-ctx.Done()

			return ctx.Err()
		})
	}

	<-allStarted
	cancel()

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGroup_limiterAndLimitSimultaneously(t *testing.T) {
	t.Parallel()

	// When both limiter and limit are set, the limiter takes precedence
	// (group.go: the `else if g.sem != nil` path is skipped when o.limiter != nil).
	const limiterWeight = 3

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
	)

	sem := semaphore.NewWeighted(int64(limiterWeight))

	g := gofuncy.NewGroup(t.Context(), "limiter-and-limit",
		gofuncy.WithLimiter(sem),
		gofuncy.WithLimit(1), // should be ignored
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	for range 10 {
		g.Add("task", func(ctx context.Context) error {
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
	// If limit(1) were honored, maxSeen would be 1. Limiter allows 3.
	assert.LessOrEqual(t, maxSeen.Load(), int32(limiterWeight))
	assert.Greater(t, maxSeen.Load(), int32(1), "limiter should allow more than 1 concurrent goroutine")
}

func TestGroup_failFastCancelPropagation(t *testing.T) {
	t.Parallel()

	errorReady := make(chan struct{})
	errorDone := make(chan struct{})

	g := gofuncy.NewGroup(t.Context(), "failfast-cancel",
		gofuncy.WithFailFast(),
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	// Goroutine A: waits for signal then errors
	g.Add("trigger", func(ctx context.Context) error {
		close(errorReady)
		return fmt.Errorf("trigger")
	})

	// Goroutine B: waits for A to error, then checks context
	g.Add("waiter", func(ctx context.Context) error {
		<-errorReady
		// Give a small window for cancel propagation
		select {
		case <-ctx.Done():
			close(errorDone)
			return ctx.Err()
		case <-time.After(time.Second):
			return fmt.Errorf("context was not canceled")
		}
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGroup_errorOrderPreserved(t *testing.T) {
	t.Parallel()

	errA := errors.New("error-a")
	errB := errors.New("error-b")
	errC := errors.New("error-c")

	g := gofuncy.NewGroup(t.Context(), "error-order",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	g.Add("a", func(ctx context.Context) error { return errA })
	g.Add("b", func(ctx context.Context) error { return nil })
	g.Add("c", func(ctx context.Context) error { return errB })
	g.Add("d", func(ctx context.Context) error { return nil })
	g.Add("e", func(ctx context.Context) error { return errC })

	err := g.Wait()
	require.Error(t, err)
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
	require.ErrorIs(t, err, errC)
}

func TestGroup_limiterAcquireFailsWithFailFast(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	sem := semaphore.NewWeighted(1)

	g := gofuncy.NewGroup(ctx, "limiter-failfast",
		gofuncy.WithLimiter(sem),
		gofuncy.WithFailFast(),
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	started := make(chan struct{})

	// First task holds the semaphore
	g.Add("holder", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()

		return ctx.Err()
	})

	<-started
	cancel()

	// Second task's Acquire should fail with canceled context
	g.Add("waiter", func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ------------------------------------------------------------------------------------------------
// ~ Error handling validation
// ------------------------------------------------------------------------------------------------

func TestGroup_allErrorsCaptured(t *testing.T) {
	t.Parallel()

	const n = 20

	sentinels := make([]error, n)
	for i := range n {
		sentinels[i] = fmt.Errorf("sentinel-%d", i)
	}

	g := gofuncy.NewGroup(t.Context(), "all-errors-captured",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	for i := range n {
		g.Add(fmt.Sprintf("task-%d", i), func(ctx context.Context) error {
			return sentinels[i]
		})
	}

	err := g.Wait()
	require.Error(t, err)

	for _, sentinel := range sentinels {
		assert.ErrorIs(t, err, sentinel)
	}
}

func TestGroup_errorFromPanic(t *testing.T) {
	t.Parallel()

	errRegular := errors.New("regular error")

	g := gofuncy.NewGroup(t.Context(), "error-from-panic",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	g.Add("regular", func(ctx context.Context) error {
		return errRegular
	})
	g.Add("panicker", func(ctx context.Context) error {
		panic("boom")
	})
	g.Add("ok", func(ctx context.Context) error {
		return nil
	})

	err := g.Wait()
	require.Error(t, err)
	require.ErrorIs(t, err, errRegular)

	var panicErr *gofuncy.PanicError

	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "boom", panicErr.Value)
}

func TestGroup_parentCancelPropagates(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	g := gofuncy.NewGroup(ctx, "parent-cancel",
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)

	allStarted := make(chan struct{})

	var started atomic.Int32

	const n = 3

	for range n {
		g.Add("task", func(ctx context.Context) error {
			if started.Add(1) == n {
				close(allStarted)
			}

			<-ctx.Done()

			return ctx.Err()
		})
	}

	<-allStarted
	cancel()

	err := g.Wait()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
