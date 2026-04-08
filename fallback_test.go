package gofuncy_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallback_noError(t *testing.T) {
	t.Parallel()

	var fallbackCalled atomic.Bool

	g := gofuncy.NewGroup(t.Context(), "fallback-noop")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		fallbackCalled.Store(true)
		return nil
	}))

	require.NoError(t, g.Wait())
	assert.False(t, fallbackCalled.Load())
}

func TestFallback_suppressesError(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context(), "fallback-suppress")
	g.Add("task", func(ctx context.Context) error {
		return fmt.Errorf("transient")
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		return nil
	}))

	require.NoError(t, g.Wait())
}

func TestFallback_replacesError(t *testing.T) {
	t.Parallel()

	replacement := fmt.Errorf("fallback error")

	g := gofuncy.NewGroup(t.Context(), "fallback-replace")
	g.Add("task", func(ctx context.Context) error {
		return fmt.Errorf("original")
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		return replacement
	}))

	err := g.Wait()
	require.ErrorIs(t, err, replacement)
}

func TestFallback_receivesOriginalError(t *testing.T) {
	t.Parallel()

	original := fmt.Errorf("original error")

	var received error

	g := gofuncy.NewGroup(t.Context(), "fallback-receives")
	g.Add("task", func(ctx context.Context) error {
		return original
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		received = err
		return nil
	}))

	require.NoError(t, g.Wait())
	assert.Equal(t, original, received)
}

func TestFallback_doesNotFallbackOnContextCanceled(t *testing.T) {
	t.Parallel()

	var fallbackCalled atomic.Bool

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	g := gofuncy.NewGroup(ctx, "fallback-ctx")
	g.Add("task", func(ctx context.Context) error {
		return context.Canceled
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		fallbackCalled.Store(true)
		return nil
	}))

	err := g.Wait()
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, fallbackCalled.Load())
}

func TestFallback_doesNotFallbackOnPanicError(t *testing.T) {
	t.Parallel()

	var fallbackCalled atomic.Bool

	g := gofuncy.NewGroup(t.Context(), "fallback-panic")
	g.Add("task", func(ctx context.Context) error {
		return &gofuncy.PanicError{Value: "boom"}
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		fallbackCalled.Store(true)
		return nil
	}))

	err := g.Wait()

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.False(t, fallbackCalled.Load())
}

func TestFallback_customFallbackIf(t *testing.T) {
	t.Parallel()

	retryable := fmt.Errorf("retryable")
	nonRetryable := fmt.Errorf("non-retryable")

	var fallbackCalled atomic.Int32

	g := gofuncy.NewGroup(t.Context(), "fallback-if")

	fallbackFn := func(ctx context.Context, err error) error {
		fallbackCalled.Add(1)
		return nil
	}

	fallbackIf := gofuncy.FallbackIf(func(err error) bool {
		return errors.Is(err, retryable)
	})

	g.Add("task1", func(ctx context.Context) error {
		return retryable
	}, gofuncy.WithFallback(fallbackFn, fallbackIf))

	g.Add("task2", func(ctx context.Context) error {
		return nonRetryable
	}, gofuncy.WithFallback(fallbackFn, fallbackIf))

	err := g.Wait()
	require.ErrorIs(t, err, nonRetryable)
	assert.Equal(t, int32(1), fallbackCalled.Load())
}

func TestFallback_composesWithRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context(), "fallback-retry")
	g.Add("task", func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("persistent")
	},
		gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))),
		gofuncy.WithFallback(func(ctx context.Context, err error) error {
			return nil // suppress after retries exhausted
		}),
	)

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(3), calls.Load())
}
