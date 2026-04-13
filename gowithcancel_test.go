package gofuncy_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleGoWithCancel() {
	var running atomic.Bool

	stop := gofuncy.GoWithCancel(context.Background(), func(ctx context.Context) error {
		running.Store(true)

		<-ctx.Done()
		running.Store(false)

		return nil
	})

	// Goroutine is running...
	time.Sleep(10 * time.Millisecond)
	fmt.Println("running:", running.Load())

	stop()
	time.Sleep(10 * time.Millisecond)
	fmt.Println("running:", running.Load())
	// Output:
	// running: true
	// running: false
}

func TestGoWithCancel_basic(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	done := make(chan struct{})

	stop := gofuncy.GoWithCancel(t.Context(), func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(done)

		return ctx.Err()
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for goroutine to start")
	}

	stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for goroutine to stop")
	}
}

func TestGoWithCancel_stopIsIdempotent(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	done := make(chan struct{})

	stop := gofuncy.GoWithCancel(t.Context(), func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(done)

		return nil
	})

	<-started

	stop()
	stop()
	stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestGoWithCancel_errorHandler(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	_ = gofuncy.GoWithCancel(t.Context(), func(ctx context.Context) error {
		return fmt.Errorf("task error")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "task error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

func TestGoWithCancel_panicRecovery(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	_ = gofuncy.GoWithCancel(t.Context(), func(ctx context.Context) error {
		panic("gowithcancel panic")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "gowithcancel panic", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestGoWithCancel_canceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	_ = gofuncy.GoWithCancel(ctx, func(ctx context.Context) error {
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
