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
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
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

	gofuncy.Go(context.Background(), func(ctx context.Context) error {
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

	gofuncy.Go(t.Context(),
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

func TestGo_withTracing(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithLogger(l),
		gofuncy.WithTracing(),
		gofuncy.WithTracerProvider(tp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withStartedCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithLogger(l),
		gofuncy.WithStartedCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withFinishedCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithLogger(l),
		gofuncy.WithFinishedCounter(),
		gofuncy.WithMeterProvider(mp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withErrorCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			return fmt.Errorf("metric error")
		},
		gofuncy.WithLogger(l),
		gofuncy.WithErrorCounter(),
		gofuncy.WithMeterProvider(mp),
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "metric error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestGo_withActiveUpDownCounter(t *testing.T) {
	t.Parallel()

	l := slog.New(slogx.NewTestHandler(t))
	mp := oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithLogger(l),
		gofuncy.WithActiveUpDownCounter(),
		gofuncy.WithMeterProvider(mp),
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

	gofuncy.Go(t.Context(),
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

func TestGo_errorHandler(t *testing.T) {
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
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

	gofuncy.Go(t.Context(),
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

	gofuncy.Go(ctx,
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

	gofuncy.Go(ctx,
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
		gofuncy.Go(t.Context(),
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
