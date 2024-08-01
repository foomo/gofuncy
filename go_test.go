package gofuncy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
)

func TestGo(t *testing.T) {
	t.Parallel()
	operation := func(ctx context.Context) error {
		return nil
	}
	errChan := gofuncy.Go(operation)
	assert.NoError(t, <-errChan)
}

func TestGoError(t *testing.T) {
	t.Parallel()
	err := errors.New("error")
	operation := func(ctx context.Context) error {
		return err
	}
	errChan := gofuncy.Go(operation)
	assert.ErrorIs(t, err, <-errChan)
}

func TestGo_WithContext(t *testing.T) {
	t.Parallel()
	operation := func(ctx context.Context) error {
		assert.Equal(t, "value", ctx.Value("key"))
		return nil
	}
	errChan := gofuncy.Go(operation, gofuncy.WithContext(context.WithValue(context.Background(), "key", "value")))
	assert.NoError(t, <-errChan)
}

func TestGo_WithName(t *testing.T) {
	t.Parallel()
	operation := func(ctx context.Context) error {
		assert.Equal(t, "gofuncy", gofuncy.RoutineFromContext(ctx))
		return nil
	}
	errChan := gofuncy.Go(operation, gofuncy.WithName("gofuncy"))
	assert.NoError(t, <-errChan)
}
