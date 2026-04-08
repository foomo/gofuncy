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

func TestDo_success(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), "ok", func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
}

func TestDo_returnsError(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), "fail", func(ctx context.Context) error {
		return fmt.Errorf("boom")
	})

	require.EqualError(t, err, "boom")
}

func TestDo_panicRecovery(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), "panic", func(ctx context.Context) error {
		panic("oops")
	})

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "oops", panicErr.Value)
}

func TestDo_withRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	err := gofuncy.Do(t.Context(), "retry", func(ctx context.Context) error {
		if calls.Add(1) < 3 {
			return fmt.Errorf("transient")
		}

		return nil
	}, gofuncy.WithRetry(5, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_withTimeout(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), "timeout", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(10*time.Millisecond))

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDo_withCircuitBreaker(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(gofuncy.CircuitBreakerThreshold(2))

	// Trip the circuit
	for range 2 {
		_ = gofuncy.Do(t.Context(), "trip", func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
	}

	// Next call should be rejected
	err := gofuncy.Do(t.Context(), "open", func(ctx context.Context) error {
		t.Fatal("should not be called")
		return nil
	}, gofuncy.WithCircuitBreaker(cb))

	require.ErrorIs(t, err, gofuncy.ErrCircuitOpen)
}

func TestDo_withFallback(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), "fallback", func(ctx context.Context) error {
		return fmt.Errorf("original")
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		return nil
	}))

	require.NoError(t, err)
}

func TestDo_fullResilienceChain(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	cb := gofuncy.NewCircuitBreaker(gofuncy.CircuitBreakerThreshold(10))

	err := gofuncy.Do(t.Context(), "full-chain", func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("fail")
	},
		gofuncy.WithTimeout(time.Second),
		gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))),
		gofuncy.WithCircuitBreaker(cb),
		gofuncy.WithFallback(func(ctx context.Context, err error) error {
			return nil // suppress after retries exhausted
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}
