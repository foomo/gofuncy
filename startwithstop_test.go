package gofuncy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleStartWithStop() {
	done := make(chan struct{})

	gofuncy.StartWithStop(context.Background(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		defer close(done)

		fmt.Println("started")
		stop() // self-cancel
		<-ctx.Done()
		fmt.Println("stopped")

		return nil
	})

	<-done
	// Output:
	// started
	// stopped
}

func TestStartWithStop_basic(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	gofuncy.StartWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		defer close(done)

		stop()
		<-ctx.Done()

		return nil
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for goroutine to stop")
	}
}

func TestStartWithStop_stopIsIdempotent(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	gofuncy.StartWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		defer close(done)

		stop()
		stop()
		stop()
		<-ctx.Done()

		return nil
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestStartWithStop_errorHandler(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.StartWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
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

func TestStartWithStop_panicRecovery(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.StartWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		panic("startwithstop panic")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "startwithstop panic", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestStartWithStop_canceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.StartWithStop(ctx, func(ctx context.Context, stop gofuncy.StopFunc) error {
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
