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

func ExampleStartWithReady() {
	proceed := make(chan struct{})
	done := make(chan struct{})

	gofuncy.StartWithReady(context.Background(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		defer close(done)

		fmt.Println("initializing")
		ready()

		<-proceed
		fmt.Println("running")

		return nil
	})

	fmt.Println("ready")
	close(proceed)
	<-done
	// Output:
	// initializing
	// ready
	// running
}

func TestStartWithReady_basic(t *testing.T) {
	t.Parallel()

	var readyCalled atomic.Bool

	done := make(chan struct{})

	gofuncy.StartWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		readyCalled.Store(true)
		ready()

		<-ctx.Done()
		close(done)

		return ctx.Err()
	})

	assert.True(t, readyCalled.Load())

	select {
	case <-done:
		t.Fatal("goroutine should still be running")
	default:
	}
}

func TestStartWithReady_noReadyCall(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.StartWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		return fmt.Errorf("init failed")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "init failed")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestStartWithReady_panicRecovery(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.StartWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		panic("before ready")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "before ready", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestStartWithReady_canceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.StartWithReady(ctx, func(ctx context.Context, ready gofuncy.ReadyFunc) error {
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

func TestStartWithReady_errorHandler(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.StartWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		ready()
		return fmt.Errorf("post-ready error")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "post-ready error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

func TestStartWithReady_multipleReadyCalls(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	gofuncy.StartWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		ready()
		ready()
		ready()
		close(done)

		return nil
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
