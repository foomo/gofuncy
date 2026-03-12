package gofuncy_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(Run(m))
}

func TestGo_withName(t *testing.T) {
	expected := "gofuncy_test"
	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			assert.Equal(t, expected, gofuncy.NameFromContext(ctx))
			return nil
		},
		gofuncy.WithName(expected),
	)
	assert.NoError(t, <-errChan)
}

func TestGo_withContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errChan := gofuncy.Go(ctx,
		func(ctx context.Context) error {
			return nil
		},
	)

	require.ErrorIs(t, <-errChan, context.Canceled)
}

func TestGo_withContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	errChan := gofuncy.Go(ctx,
		func(ctx context.Context) error {
			cancel()
			return ctx.Err()
		},
	)

	require.ErrorIs(t, <-errChan, context.Canceled)
}

func TestGo_withNilOption(t *testing.T) {
	var called atomic.Bool

	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		nil, // passing nil option should not panic
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestGo_withDurationMetric(t *testing.T) {
	var called atomic.Bool

	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		gofuncy.WithDurationMetric(),
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestGo_contextNamePropagation(t *testing.T) {
	parentName := "parent-routine"

	var childName string

	var mu sync.Mutex

	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			// Inside the routine, spawn a child routine
			childErrChan := gofuncy.Go(ctx,
				func(childCtx context.Context) error {
					mu.Lock()
					childName = gofuncy.NameFromContext(childCtx)
					mu.Unlock()

					return nil
				},
				gofuncy.WithName("child-routine"),
			)

			return <-childErrChan
		},
		gofuncy.WithName(parentName),
	)

	err := <-errChan
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, "child-routine", childName)
	mu.Unlock()
}

func TestGo_concurrent(t *testing.T) {
	const numGoroutines = 100

	var wg sync.WaitGroup

	var successCount atomic.Int32

	var errCount atomic.Int32

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()

			errChan := gofuncy.Go(t.Context(),
				func(ctx context.Context) error {
					return nil
				},
				gofuncy.WithName(fmt.Sprintf("goroutine-%d", idx)),
			)

			if err := <-errChan; err != nil {
				errCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(numGoroutines), successCount.Load())
	assert.Equal(t, int32(0), errCount.Load())
}
