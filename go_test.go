package gofuncy_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	slogx "github.com/foomo/go/slog"
	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossymetric"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/semaphore"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func ExampleGo() {
	done := make(chan struct{})

	gofuncy.Go(context.Background(), "example", func(ctx context.Context) error {
		defer close(done)

		fmt.Println("running")

		return nil
	})

	<-done
	// Output:
	// running
}

func TestGo_basic(t *testing.T) {
	done := make(chan struct{})

	gofuncy.Go(t.Context(), "basic",
		func(ctx context.Context) error {
			close(done)
			return nil
		},
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withDurationHistogram(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(), "duration-histogram",
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithLogger(l),
		gofuncy.WithDurationHistogram(),
		gofuncy.WithMeterProvider(mp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withStallThreshold(t *testing.T) {
	t.Parallel()

	stallCh := make(chan struct{}, 1)
	done := make(chan struct{})

	gofuncy.Go(t.Context(), "stall-test",
		func(ctx context.Context) error {
			<-stallCh
			close(done)

			return nil
		},
		gofuncy.WithStallThreshold(10*time.Millisecond),
		gofuncy.WithStallHandler(func(ctx context.Context, name string, elapsed time.Duration) {
			stallCh <- struct{}{}
		}),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stall handler to unblock goroutine")
	}
}

func TestGo_errorHandler(t *testing.T) {
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(), "error-handler",
		func(ctx context.Context) error {
			return fmt.Errorf("test error")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "test error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

func TestGo_panicRecovery(t *testing.T) {
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(), "panic-recovery",
		func(ctx context.Context) error {
			panic("fire and forget panic")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "fire and forget panic", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestGo_canceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.Go(ctx, "canceled-ctx",
		func(ctx context.Context) error {
			return nil
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for context error")
	}
}

func TestGo_contextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)

	gofuncy.Go(ctx, "ctx-canceled",
		func(ctx context.Context) error {
			cancel()
			return ctx.Err()
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for context error")
	}
}

func TestGo_withLimiter(t *testing.T) {
	t.Parallel()

	const (
		limit = 2
		total = 8
	)

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
		wg      sync.WaitGroup
	)

	sem := semaphore.NewWeighted(int64(limit))

	wg.Add(total)

	for range total {
		gofuncy.Go(t.Context(), "limiter-test",
			func(ctx context.Context) error {
				defer wg.Done()

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
			},
			gofuncy.WithLimiter(sem),
		)
	}

	wg.Wait()
	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))
}

// ------------------------------------------------------------------------------------------------
// ~ Edge cases
// ------------------------------------------------------------------------------------------------

func TestGo_withTimeout(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(), "timeout-test",
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		gofuncy.WithTimeout(50*time.Millisecond),
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for deadline exceeded error")
	}
}

func TestGo_withZeroTimeout(t *testing.T) {
	t.Parallel()

	// WithTimeout(0) is a no-op because the guard is `o.timeout > 0`.
	// The function should complete normally.
	done := make(chan struct{})

	gofuncy.Go(t.Context(), "zero-timeout",
		func(ctx context.Context) error {
			require.NoError(t, ctx.Err(), "context should not be canceled with zero timeout")
			close(done)

			return nil
		},
		gofuncy.WithTimeout(0),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_nilFunction(t *testing.T) {
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(), "nil-fn",
		nil,
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error from nil function")
	}
}

func TestGo_concurrentWithSharedLimiter(t *testing.T) {
	t.Parallel()

	const total = 20

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
		wg      sync.WaitGroup
	)

	sem := semaphore.NewWeighted(1)

	wg.Add(total)

	for range total {
		gofuncy.Go(t.Context(), "shared-limiter",
			func(ctx context.Context) error {
				defer wg.Done()

				cur := active.Add(1)

				for {
					old := maxSeen.Load()
					if cur <= old || maxSeen.CompareAndSwap(old, cur) {
						break
					}
				}

				time.Sleep(time.Millisecond)
				active.Add(-1)

				return nil
			},
			gofuncy.WithLimiter(sem),
		)
	}

	wg.Wait()
	assert.Equal(t, int32(1), maxSeen.Load())
}

func TestGo_contextValuePreservation(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}

	ctx := context.WithValue(t.Context(), ctxKey{}, "preserved")
	done := make(chan struct{})

	gofuncy.Go(ctx, "ctx-values",
		func(ctx context.Context) error {
			assert.Equal(t, "preserved", ctx.Value(ctxKey{}))
			close(done)

			return nil
		},
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_errorHandlerReceivesDeadlineExceeded(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(), "deadline-exceeded",
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		gofuncy.WithTimeout(time.Millisecond),
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Error handling & context validation
// ------------------------------------------------------------------------------------------------

func TestGo_limiterAcquireFailsOnCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)
	spawned := make(chan struct{})

	sem := semaphore.NewWeighted(1)

	gofuncy.Go(ctx, "limiter-canceled",
		func(ctx context.Context) error {
			close(spawned)
			return nil
		},
		gofuncy.WithLimiter(sem),
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for limiter acquire error")
	}

	select {
	case <-spawned:
		t.Fatal("goroutine should not have been spawned")
	default:
	}
}

func TestGo_contextValuesAccessibleInErrorHandler(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}

	ctx := context.WithValue(t.Context(), ctxKey{}, "available")
	done := make(chan struct{})

	gofuncy.Go(ctx, "ctx-values-err",
		func(ctx context.Context) error {
			return fmt.Errorf("test error")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			assert.Equal(t, "available", ctx.Value(ctxKey{}))
			close(done)
		}),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}
