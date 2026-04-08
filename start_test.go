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
)

func TestStart_success(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "ok", func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, wait())
}

func TestStart_returnsError(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "fail", func(ctx context.Context) error {
		return fmt.Errorf("boom")
	})

	require.EqualError(t, wait(), "boom")
}

func TestStart_panicRecovery(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "panic", func(ctx context.Context) error {
		panic("oops")
	})

	err := wait()

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "oops", panicErr.Value)
}

func TestStart_withRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	wait := gofuncy.Start(t.Context(), "retry", func(ctx context.Context) error {
		if calls.Add(1) < 3 {
			return fmt.Errorf("transient")
		}

		return nil
	}, gofuncy.WithRetry(5, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))))

	require.NoError(t, wait())
	assert.Equal(t, int32(3), calls.Load())
}

func TestStart_withTimeout(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "timeout", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(10*time.Millisecond))

	require.ErrorIs(t, wait(), context.DeadlineExceeded)
}

func TestStart_withFallback(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "fallback", func(ctx context.Context) error {
		return fmt.Errorf("original")
	}, gofuncy.WithFallback(func(ctx context.Context, err error) error {
		return nil
	}))

	require.NoError(t, wait())
}

func TestStart_multipleWaitCalls(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "multi-wait", func(ctx context.Context) error {
		return fmt.Errorf("fail")
	})

	err1 := wait()
	err2 := wait()
	err3 := wait()

	require.EqualError(t, err1, "fail")
	assert.Equal(t, err1, err2)
	assert.Equal(t, err2, err3)
}

func TestStart_concurrentWaiters(t *testing.T) {
	t.Parallel()

	wait := gofuncy.Start(t.Context(), "concurrent", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return fmt.Errorf("done")
	})

	var wg sync.WaitGroup

	for range 5 {
		wg.Go(func() {
			err := wait()
			assert.EqualError(t, err, "done")
		})
	}

	wg.Wait()
}

func TestStart_doWorkBeforeWait(t *testing.T) {
	t.Parallel()

	var order []string

	var mu sync.Mutex

	wait := gofuncy.Start(t.Context(), "async", func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()

		order = append(order, "async")
		mu.Unlock()

		return nil
	})

	// Do work while async runs
	mu.Lock()

	order = append(order, "sync")
	mu.Unlock()

	require.NoError(t, wait())

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, []string{"sync", "async"}, order)
}
