package gofuncy_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleMap() {
	results, err := gofuncy.Map(
		context.Background(),
		[]int{1, 2, 3},
		func(ctx context.Context, v int) (string, error) {
			return strconv.Itoa(v * 2), nil
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(results)
	// Output:
	// [2 4 6]
}

func ExampleMapBackground() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — goroutines still run

	results, err := gofuncy.MapBackground(
		ctx,
		[]int{1, 2, 3},
		func(ctx context.Context, v int) (string, error) {
			return strconv.Itoa(v * 2), nil
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(results)
	// Output:
	// [2 4 6]
}

func TestMap_basic(t *testing.T) {
	t.Parallel()

	input := []int{1, 2, 3, 4, 5}

	results, err := gofuncy.Map(t.Context(), input,
		func(ctx context.Context, v int) (string, error) {
			return strconv.Itoa(v * 2), nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"2", "4", "6", "8", "10"}, results)
}

func TestMap_empty(t *testing.T) {
	t.Parallel()

	results, err := gofuncy.Map(t.Context(), []int{},
		func(ctx context.Context, v int) (int, error) {
			return v, nil
		},
	)

	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestMap_preservesOrder(t *testing.T) {
	t.Parallel()

	input := make([]int, 100)
	for i := range input {
		input[i] = i
	}

	results, err := gofuncy.Map(t.Context(), input,
		func(ctx context.Context, v int) (int, error) {
			return v * 3, nil
		},
	)

	require.NoError(t, err)

	for i, r := range results {
		assert.Equal(t, i*3, r, "index %d", i)
	}
}

func TestMap_errors(t *testing.T) {
	t.Parallel()

	errBad := errors.New("bad value")

	results, err := gofuncy.Map(t.Context(), []int{1, 2, 3},
		func(ctx context.Context, v int) (int, error) {
			if v == 2 {
				return 0, errBad
			}

			return v * 10, nil
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errBad)
	// successful results should still be populated
	assert.Equal(t, 10, results[0])
	assert.Equal(t, 30, results[2])
}

func TestMap_failFast(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("fail fast")

	_, err := gofuncy.Map(t.Context(), []int{1, 2, 3, 4, 5},
		func(ctx context.Context, v int) (int, error) {
			if v == 1 {
				return 0, sentinel
			}

			<-ctx.Done()

			return 0, ctx.Err()
		},
		gofuncy.WithFailFast(),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
}

func TestMap_withLimit(t *testing.T) {
	t.Parallel()

	const (
		limit = 2
		total = 20
	)

	var (
		running    atomic.Int32
		maxRunning atomic.Int32
	)

	input := make([]int, total)
	for i := range input {
		input[i] = i
	}

	results, err := gofuncy.Map(t.Context(), input,
		func(ctx context.Context, v int) (int, error) {
			cur := running.Add(1)

			for {
				old := maxRunning.Load()
				if cur <= old || maxRunning.CompareAndSwap(old, cur) {
					break
				}
			}

			running.Add(-1)

			return v * 2, nil
		},
		gofuncy.WithLimit(limit),
	)

	require.NoError(t, err)
	assert.LessOrEqual(t, maxRunning.Load(), int32(limit))

	for i, r := range results {
		assert.Equal(t, i*2, r)
	}
}

func TestMapBackground_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	results, err := gofuncy.MapBackground(ctx, []int{1, 2, 3},
		func(ctx context.Context, v int) (int, error) {
			return v * 10, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []int{10, 20, 30}, results)
}

func TestMap_panicRecovery(t *testing.T) {
	t.Parallel()

	results, err := gofuncy.Map(t.Context(), []int{1, 2, 3},
		func(ctx context.Context, v int) (int, error) {
			if v == 2 {
				panic("map panic")
			}

			return v * 10, nil
		},
	)

	require.Error(t, err)

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "map panic", panicErr.Value)
	// other results should still be populated
	assert.Equal(t, 10, results[0])
	assert.Equal(t, 30, results[2])
}
