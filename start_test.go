package gofuncy_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func ExampleStart() {
	done := make(chan struct{})

	gofuncy.Start(context.Background(), func(ctx context.Context) error {
		defer close(done)

		fmt.Println("running")

		return nil
	})

	<-done
	// Output:
	// running
}

func TestStart_goroutineIsRunning(t *testing.T) {
	t.Parallel()

	var running atomic.Bool

	done := make(chan struct{})

	gofuncy.Start(t.Context(), func(ctx context.Context) error {
		running.Store(true)
		close(done)

		return nil
	})

	// The goroutine must have started by the time Start returns.
	select {
	case <-done:
		assert.True(t, running.Load())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for goroutine to complete")
	}
}

func TestStart_panicRecovery(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Start(t.Context(),
		func(ctx context.Context) error {
			panic("start panic")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "start panic", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestStart_errorHandler(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Start(t.Context(),
		func(ctx context.Context) error {
			return fmt.Errorf("start error")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "start error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

func TestStart_canceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.Start(ctx,
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

func TestStart_withLimiter(t *testing.T) {
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
		gofuncy.Start(t.Context(),
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

func TestStart_limiterAcquireFailsOnCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	sem := semaphore.NewWeighted(1)

	gofuncy.Start(ctx,
		func(ctx context.Context) error {
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
}
