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

func ExampleGroup() {
	err := gofuncy.Group(context.Background(), []gofuncy.Func{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	})

	fmt.Println(err)
	// Output:
	// <nil>
}

func ExampleGroupBackground() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — goroutines still run

	err := gofuncy.GroupBackground(ctx, []gofuncy.Func{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	})

	fmt.Println(err)
	// Output:
	// <nil>
}

func TestGroup_basic(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	err := gofuncy.Group(t.Context(), []gofuncy.Func{
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), count.Load())
}

func TestGroup_empty(t *testing.T) {
	t.Parallel()

	err := gofuncy.Group(t.Context(), nil)
	require.NoError(t, err)
}

func TestGroup_errors(t *testing.T) {
	t.Parallel()

	errA := errors.New("error a")
	errB := errors.New("error b")

	err := gofuncy.Group(t.Context(), []gofuncy.Func{
		func(ctx context.Context) error { return errA },
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return errB },
	})

	require.Error(t, err)
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
}

func TestGroup_failFast(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	sentinel := errors.New("fail fast")

	err := gofuncy.Group(t.Context(), []gofuncy.Func{
		func(ctx context.Context) error {
			return sentinel
		},
		func(ctx context.Context) error {
			close(started)
			<-ctx.Done()

			return ctx.Err()
		},
	}, gofuncy.WithFailFast())

	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
}

func TestGroup_withLimit(t *testing.T) {
	t.Parallel()

	const (
		limit = 3
		total = 20
	)

	var (
		running    atomic.Int32
		maxRunning atomic.Int32
	)

	fns := make([]gofuncy.Func, total)
	for i := range total {
		fns[i] = func(ctx context.Context) error {
			cur := running.Add(1)
			// update max running
			for {
				old := maxRunning.Load()
				if cur <= old || maxRunning.CompareAndSwap(old, cur) {
					break
				}
			}

			// do some work
			running.Add(-1)

			return nil
		}
	}

	err := gofuncy.Group(t.Context(), fns, gofuncy.WithLimit(limit))
	require.NoError(t, err)
	assert.LessOrEqual(t, maxRunning.Load(), int32(limit))
}

func TestGroup_panicRecovery(t *testing.T) {
	t.Parallel()

	err := gofuncy.Group(t.Context(), []gofuncy.Func{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { panic("group panic") },
		func(ctx context.Context) error { return nil },
	})

	require.Error(t, err)

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "group panic", panicErr.Value)
}

func TestGroup_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := gofuncy.Group(ctx, []gofuncy.Func{
		func(ctx context.Context) error {
			return ctx.Err()
		},
	})

	require.ErrorIs(t, err, context.Canceled)
}

func TestGroupBackground_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var count atomic.Int32

	err := gofuncy.GroupBackground(ctx, []gofuncy.Func{
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, int32(2), count.Load())
}

func TestGroup_concurrent(t *testing.T) {
	t.Parallel()

	const n = 100

	fns := make([]gofuncy.Func, n)
	results := make([]int, n)

	for i := range n {
		fns[i] = func(ctx context.Context) error {
			results[i] = i * 2
			return nil
		}
	}

	err := gofuncy.Group(t.Context(), fns)
	require.NoError(t, err)

	for i := range n {
		assert.Equal(t, i*2, results[i], "index %d", i)
	}
}
