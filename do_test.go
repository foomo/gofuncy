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

func ExampleDo() {
	err := gofuncy.Do(context.Background(), func(ctx context.Context) error {
		fmt.Println("hello")
		return nil
	})

	fmt.Println("error:", err)
	// Output:
	// hello
	// error: <nil>
}

func ExampleDo_retry() {
	var attempts atomic.Int32

	err := gofuncy.Do(context.Background(),
		func(ctx context.Context) error {
			n := attempts.Add(1)
			if n < 3 {
				return fmt.Errorf("attempt %d failed", n)
			}

			fmt.Println("succeeded on attempt", n)

			return nil
		},
		gofuncy.WithRetry(5),
		gofuncy.WithTimeout(time.Second),
	)

	fmt.Println("error:", err)
	// Output:
	// succeeded on attempt 3
	// error: <nil>
}

func ExampleDo_fallback() {
	err := gofuncy.Do(context.Background(),
		func(ctx context.Context) error {
			return fmt.Errorf("primary failed")
		},
		gofuncy.WithFallback(func(ctx context.Context, err error) error {
			fmt.Println("fallback called:", err)
			return nil // suppress the error
		}),
	)

	fmt.Println("error:", err)
	// Output:
	// fallback called: primary failed
	// error: <nil>
}

func TestDo_success(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
}

func TestDo_returnsError(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
		return fmt.Errorf("boom")
	})

	require.EqualError(t, err, "boom")
}

func TestDo_panicRecovery(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
		panic("oops")
	})

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "oops", panicErr.Value)
}

func TestDo_withRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
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

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
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
		_ = gofuncy.Do(t.Context(), func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
	}

	// Next call should be rejected
	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
		t.Fatal("should not be called")
		return nil
	}, gofuncy.WithCircuitBreaker(cb))

	require.ErrorIs(t, err, gofuncy.ErrCircuitOpen)
}

func TestDo_withFallback(t *testing.T) {
	t.Parallel()

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
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

	err := gofuncy.Do(t.Context(), func(ctx context.Context) error {
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
