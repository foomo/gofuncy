package gofuncy_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossymetric"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleAsync() {
	errCh := gofuncy.Async(context.Background(), func(ctx context.Context) error {
		return nil
	})

	fmt.Println(<-errCh)
	// Output:
	// <nil>
}

func ExampleAsyncBackground() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — goroutine still runs

	errCh := gofuncy.AsyncBackground(ctx, func(ctx context.Context) error {
		return nil
	})

	fmt.Println(<-errCh)
	// Output:
	// <nil>
}

func TestAsync_withName(t *testing.T) {
	expected := "gofuncy_test"
	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			assert.Equal(t, expected, gofuncy.NameFromContext(ctx))
			return nil
		},
		gofuncy.AsyncOption().WithName(expected),
	)
	assert.NoError(t, <-errChan)
}

func TestAsync_withContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errChan := gofuncy.Async(ctx,
		func(ctx context.Context) error {
			return nil
		},
	)

	require.ErrorIs(t, <-errChan, context.Canceled)
}

func TestAsync_withContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	errChan := gofuncy.Async(ctx,
		func(ctx context.Context) error {
			cancel()
			return ctx.Err()
		},
	)

	require.ErrorIs(t, <-errChan, context.Canceled)
}

func TestAsync_withNilOption(t *testing.T) {
	var called atomic.Bool

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		nil, // passing nil builder should not panic
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestAsync_withTracing(t *testing.T) {
	oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	var called atomic.Bool

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		gofuncy.AsyncOption().WithTracing(),
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestAsync_withCounterMetric(t *testing.T) {
	oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	var called atomic.Bool

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		gofuncy.AsyncOption().WithCounterMetric(),
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestAsync_withUpDownMetric(t *testing.T) {
	oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	var called atomic.Bool

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		gofuncy.AsyncOption().WithUpDownMetric(),
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestAsync_withDurationMetric(t *testing.T) {
	oteltesting.ReportMetrics(t, glossymetric.NewTest(t))

	var called atomic.Bool

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			time.Sleep(time.Second)

			return nil
		},
		gofuncy.AsyncOption().WithDurationMetric(),
	)

	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestAsync_contextNamePropagation(t *testing.T) {
	parentName := "parent-routine"

	var childName string

	var mu sync.Mutex

	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			// Inside the routine, spawn a child routine
			childErrChan := gofuncy.Async(ctx,
				func(childCtx context.Context) error {
					mu.Lock()
					childName = gofuncy.NameFromContext(childCtx)
					mu.Unlock()

					return nil
				},
				gofuncy.AsyncOption().WithName("child-routine"),
			)

			return <-childErrChan
		},
		gofuncy.AsyncOption().WithName(parentName),
	)

	err := <-errChan
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, "child-routine", childName)
	mu.Unlock()
}

func TestAsync_concurrent(t *testing.T) {
	const numGoroutines = 100

	var wg sync.WaitGroup

	var successCount atomic.Int32

	var errCount atomic.Int32

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()

			errChan := gofuncy.Async(t.Context(),
				func(ctx context.Context) error {
					return nil
				},
				gofuncy.AsyncOption().WithName(fmt.Sprintf("goroutine-%d", idx)),
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

func TestAsync_panicRecovery(t *testing.T) {
	errChan := gofuncy.Async(t.Context(),
		func(ctx context.Context) error {
			panic("test panic")
		},
	)

	err := <-errChan
	require.Error(t, err)

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "test panic", panicErr.Value)
	assert.NotEmpty(t, panicErr.Stack)
}
