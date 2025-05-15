package gofuncy_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

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

func TestGo_Error(t *testing.T) {
	t.Parallel()
	expected := errors.New("error")
	errChan := gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			return expected
		},
	)
	assert.ErrorIs(t, expected, <-errChan)
}

func TestGo_WithName(t *testing.T) {
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
