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

func TestAll_basic(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	items := []int{1, 2, 3, 4, 5}

	err := gofuncy.All(t.Context(), "basic", items, func(ctx context.Context, item int) error {
		count.Add(1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(5), count.Load())
}

func TestAll_empty(t *testing.T) {
	t.Parallel()

	err := gofuncy.All(t.Context(), "empty", []int{}, func(ctx context.Context, item int) error {
		t.Fatal("should not be called")
		return nil
	})

	require.NoError(t, err)
}

func TestAll_errors(t *testing.T) {
	t.Parallel()

	errA := errors.New("error a")
	errB := errors.New("error b")

	items := []int{1, 2, 3}

	err := gofuncy.All(t.Context(), "with-errors", items, func(ctx context.Context, item int) error {
		switch item {
		case 1:
			return errA
		case 3:
			return errB
		default:
			return nil
		}
	})

	require.Error(t, err)
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
}

func TestAll_failFast(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})

	err := gofuncy.All(t.Context(), "fail-fast", []int{1, 2},
		func(ctx context.Context, item int) error {
			if item == 1 {
				close(started)
				return fmt.Errorf("first error")
			}

			<-started
			<-ctx.Done()

			return ctx.Err()
		},
		gofuncy.WithFailFast(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAll_withLimit(t *testing.T) {
	t.Parallel()

	const limit = 2

	var (
		active  atomic.Int32
		maxSeen atomic.Int32
	)

	items := make([]int, 10)
	for i := range items {
		items[i] = i
	}

	err := gofuncy.All(t.Context(), "with-limit", items,
		func(ctx context.Context, item int) error {
			cur := active.Add(1)

			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			active.Add(-1)

			return nil
		},
		gofuncy.WithLimit(limit),
	)

	require.NoError(t, err)
	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))
}

func TestAll_singleItem(t *testing.T) {
	t.Parallel()

	var got int

	err := gofuncy.All(t.Context(), "single", []int{42}, func(ctx context.Context, item int) error {
		got = item
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, got)
}

func TestAll_panic(t *testing.T) {
	t.Parallel()

	err := gofuncy.All(t.Context(), "panic", []int{1}, func(ctx context.Context, item int) error {
		panic("foreach panic")
	})

	require.Error(t, err)

	var panicErr *gofuncy.PanicError

	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "foreach panic", panicErr.Value)
}

func TestAll_contextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	started := make(chan struct{})

	err := gofuncy.All(ctx, "ctx-cancel", []int{1, 2},
		func(ctx context.Context, item int) error {
			if item == 1 {
				close(started)
				cancel()

				return ctx.Err()
			}

			<-started
			<-ctx.Done()

			return ctx.Err()
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
