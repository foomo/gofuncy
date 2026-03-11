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
	"go.uber.org/zap/zaptest"
)

func ExampleGo() {
	ctx := gofuncy.Ctx(context.Background()).Root()

	errChan := gofuncy.Go(ctx, func(ctx context.Context) error {
		fmt.Println("hello")
		return nil
	})

	if err := <-errChan; err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok")
	}

	// Output:
	// hello
	// ok
}

func ExampleGo_error() {
	ctx := gofuncy.Ctx(context.Background()).Root()

	errChan := gofuncy.Go(ctx, func(ctx context.Context) error {
		return errors.New("sth went wrong")
	})

	if err := <-errChan; err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok")
	}

	// Output:
	// sth went wrong
}

func TestGo(t *testing.T) {
	t.Parallel()
	var called atomic.Bool
	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
		gofuncy.WithLogger(zaptest.NewLogger(t)),
	)
	require.NoError(t, <-errChan)
	assert.True(t, called.Load())
}

func TestGo_error(t *testing.T) {
	t.Parallel()

	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			return errors.New("ups")
		},
	)

	err, ok := <-errChan
	assert.True(t, ok)
	require.Error(t, err)

	err, ok = <-errChan
	assert.False(t, ok)
	require.NoError(t, err)
}

func TestGo_withName(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

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
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	errChan := gofuncy.Go(ctx,
		func(ctx context.Context) error {
			cancel()
			return ctx.Err()
		},
	)

	require.ErrorIs(t, <-errChan, context.Canceled)
}
