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

func TestMap_basic(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3, 4, 5}

	results, err := gofuncy.Map(t.Context(), "double", items, func(ctx context.Context, item int) (int, error) {
		return item * 2, nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}

func TestMap_preservesOrder(t *testing.T) {
	t.Parallel()

	items := []int{5, 4, 3, 2, 1}

	results, err := gofuncy.Map(t.Context(), "format", items, func(ctx context.Context, item int) (string, error) {
		// sleep proportional to item to scramble completion order
		time.Sleep(time.Duration(item) * time.Millisecond)

		return fmt.Sprintf("item-%d", item), nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"item-5", "item-4", "item-3", "item-2", "item-1"}, results)
}

func TestMap_empty(t *testing.T) {
	t.Parallel()

	results, err := gofuncy.Map(t.Context(), "empty", []int{}, func(ctx context.Context, item int) (int, error) {
		t.Fatal("should not be called")
		return 0, nil
	})

	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestMap_errors(t *testing.T) {
	t.Parallel()

	items := []int{1, 2, 3}

	results, err := gofuncy.Map(t.Context(), "with-errors", items, func(ctx context.Context, item int) (int, error) {
		if item == 2 {
			return 0, fmt.Errorf("bad item")
		}

		return item * 10, nil
	})

	require.Error(t, err)
	// successful items should still have their results
	assert.Equal(t, 10, results[0])
	assert.Equal(t, 30, results[2])
}

func TestMap_failFast(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})

	_, err := gofuncy.Map(t.Context(), "fail-fast", []int{1, 2},
		func(ctx context.Context, item int) (int, error) {
			if item == 1 {
				close(started)

				return 0, fmt.Errorf("first error")
			}

			<-started
			<-ctx.Done()

			return 0, ctx.Err()
		},
		gofuncy.WithFailFast(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMap_withLimit(t *testing.T) {
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

	results, err := gofuncy.Map(t.Context(), "with-limit", items,
		func(ctx context.Context, item int) (int, error) {
			cur := active.Add(1)

			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			active.Add(-1)

			return item * 2, nil
		},
		gofuncy.WithLimit(limit),
	)

	require.NoError(t, err)
	assert.LessOrEqual(t, maxSeen.Load(), int32(limit))

	for i, item := range items {
		assert.Equal(t, item*2, results[i])
	}
}

func TestMap_singleItem(t *testing.T) {
	t.Parallel()

	results, err := gofuncy.Map(t.Context(), "single", []int{7}, func(ctx context.Context, item int) (string, error) {
		return fmt.Sprintf("x%d", item), nil
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "x7", results[0])
}

func TestMap_partialErrorsPreserveResults(t *testing.T) {
	t.Parallel()

	items := []int{0, 1, 2, 3, 4}

	results, err := gofuncy.Map(t.Context(), "partial-errors", items, func(ctx context.Context, item int) (int, error) {
		if item == 1 || item == 3 {
			return 0, fmt.Errorf("bad item %d", item)
		}

		return item * 10, nil
	})

	require.Error(t, err)
	require.Len(t, results, 5)
	assert.Equal(t, 0, results[0])
	assert.Equal(t, 0, results[1]) // errored — zero value
	assert.Equal(t, 20, results[2])
	assert.Equal(t, 0, results[3]) // errored — zero value
	assert.Equal(t, 40, results[4])
}

func TestMap_panic(t *testing.T) {
	t.Parallel()

	results, err := gofuncy.Map(t.Context(), "panic", []int{1, 2, 3}, func(ctx context.Context, item int) (int, error) {
		if item == 2 {
			panic("map panic")
		}

		return item * 10, nil
	})

	require.Error(t, err)

	var panicErr *gofuncy.PanicError

	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "map panic", panicErr.Value)

	// Non-panicking items should still have results
	assert.Equal(t, 10, results[0])
	assert.Equal(t, 30, results[2])
}
