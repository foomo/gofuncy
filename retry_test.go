package gofuncy_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_succeedsFirstAttempt(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}, gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetry_failsThenSucceeds(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		if calls.Add(1) < 3 {
			return fmt.Errorf("transient error")
		}

		return nil
	}, gofuncy.WithRetry(5, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetry_exhaustsAttempts(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("persistent error")
	}, gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	err := g.Wait()
	require.EqualError(t, err, "persistent error")
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetry_respectsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	var calls atomic.Int32

	g := gofuncy.NewGroup(ctx)
	g.Add(func(ctx context.Context) error {
		if calls.Add(1) == 2 {
			cancel()
		}

		return fmt.Errorf("keep going")
	}, gofuncy.WithRetry(10, gofuncy.RetryBackoff(gofuncy.BackoffConstant(time.Millisecond))))

	err := g.Wait()
	require.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, calls.Load(), int32(3))
}

func TestRetry_doesNotRetryPanicError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return &gofuncy.PanicError{Value: "boom"}
	}, gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	err := g.Wait()

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetry_doesNotRetryDeadlineExceeded(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return context.DeadlineExceeded
	}, gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	err := g.Wait()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetry_customRetryIf(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	retryable := fmt.Errorf("retryable")
	nonRetryable := fmt.Errorf("non-retryable")

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return retryable
		}

		return nonRetryable
	}, gofuncy.WithRetry(5,
		gofuncy.RetryBackoff(gofuncy.BackoffConstant(0)),
		gofuncy.RetryIf(func(err error) bool {
			return errors.Is(err, retryable)
		}),
	))

	err := g.Wait()
	require.ErrorIs(t, err, nonRetryable)
	assert.Equal(t, int32(2), calls.Load())
}

func TestRetry_customBackoff(t *testing.T) {
	t.Parallel()

	start := time.Now()

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, gofuncy.WithRetry(3,
		gofuncy.RetryBackoff(gofuncy.BackoffConstant(20*time.Millisecond)),
	))

	_ = g.Wait()
	elapsed := time.Since(start)

	// 2 delays of 20ms each (between attempts 1-2 and 2-3)
	assert.GreaterOrEqual(t, elapsed, 35*time.Millisecond)
}

func TestRetry_onRetryCallback(t *testing.T) {
	t.Parallel()

	var retryAttempts []int

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, gofuncy.WithRetry(4,
		gofuncy.RetryBackoff(gofuncy.BackoffConstant(0)),
		gofuncy.RetryOnRetry(func(ctx context.Context, attempt int, err error) {
			retryAttempts = append(retryAttempts, attempt)
		}),
	))

	_ = g.Wait()
	// OnRetry fires before retries 1, 2, 3 (not before the last attempt since it breaks)
	assert.Equal(t, []int{1, 2, 3}, retryAttempts)
}

func TestRetry_composesWithTimeout(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	// Per-attempt timeout of 20ms — each slow attempt is killed, but retry continues
	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			// First two attempts: block until per-attempt timeout
			<-ctx.Done()
			return ctx.Err()
		}
		// Third attempt: succeed
		return nil
	},
		gofuncy.WithTimeout(20*time.Millisecond),
		gofuncy.WithRetry(3,
			gofuncy.RetryBackoff(gofuncy.BackoffConstant(0)),
			gofuncy.RetryIf(func(err error) bool { return true }),
		),
	)

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(3), calls.Load())
}

func TestRetry_composesWithFailFast(t *testing.T) {
	t.Parallel()

	var retryCalls atomic.Int32

	started := make(chan struct{})

	g := gofuncy.NewGroup(t.Context(), gofuncy.WithFailFast())

	// This task retries forever until context is cancelled
	g.Add(func(ctx context.Context) error {
		if retryCalls.Add(1) == 1 {
			close(started)
		}

		return fmt.Errorf("transient")
	}, gofuncy.WithRetry(100, gofuncy.RetryBackoff(gofuncy.BackoffConstant(5*time.Millisecond))))

	// This task waits for the other to start, then fails immediately
	g.Add(func(ctx context.Context) error {
		<-started
		return fmt.Errorf("fatal error")
	})

	err := g.Wait()
	require.Error(t, err)
	// Retry task should have been stopped by fail-fast cancellation
	assert.Less(t, retryCalls.Load(), int32(100))
}

func TestRetry_maxAttemptsOne(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("fail")
	}, gofuncy.WithRetry(1))

	err := g.Wait()
	require.EqualError(t, err, "fail")
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetry_maxAttemptsZeroDefaultsToOne(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("fail")
	}, gofuncy.WithRetry(0))

	err := g.Wait()
	require.EqualError(t, err, "fail")
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetry_viaMiddleware(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		if calls.Add(1) < 3 {
			return fmt.Errorf("transient")
		}

		return nil
	}, gofuncy.WithMiddleware(gofuncy.Retry(5, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0)))))

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(3), calls.Load())
}

func TestBackoffExponential(t *testing.T) {
	t.Parallel()

	b := gofuncy.BackoffExponential(100*time.Millisecond, 2, 5*time.Second)

	// Verify exponential growth with jitter — run multiple times to check bounds
	for range 20 {
		d0 := b(0)
		d1 := b(1)
		d2 := b(2)
		d10 := b(10)

		// attempt 0: ~100ms +/- 25%
		assert.GreaterOrEqual(t, d0, 75*time.Millisecond)
		assert.LessOrEqual(t, d0, 125*time.Millisecond)

		// attempt 1: ~200ms +/- 25%
		assert.GreaterOrEqual(t, d1, 150*time.Millisecond)
		assert.LessOrEqual(t, d1, 250*time.Millisecond)

		// attempt 2: ~400ms +/- 25%
		assert.GreaterOrEqual(t, d2, 300*time.Millisecond)
		assert.LessOrEqual(t, d2, 500*time.Millisecond)

		// attempt 10: should be capped at ~5s +/- 25%
		assert.LessOrEqual(t, d10, 6250*time.Millisecond)
	}
}

func TestBackoffConstant(t *testing.T) {
	t.Parallel()

	b := gofuncy.BackoffConstant(50 * time.Millisecond)
	assert.Equal(t, 50*time.Millisecond, b(0))
	assert.Equal(t, 50*time.Millisecond, b(5))
	assert.Equal(t, 50*time.Millisecond, b(100))
}
