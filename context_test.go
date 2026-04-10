package gofuncy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleCtx() {
	done := make(chan struct{})

	gofuncy.Go(context.Background(), "worker", func(ctx context.Context) error {
		defer close(done)

		fmt.Println("name:", gofuncy.NameFromContext(ctx))

		return nil
	})

	<-done
	// Output:
	// name: worker
}

func TestCtx_NoName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, gofuncy.NameNoName, gofuncy.Ctx(t.Context()).Name())
}

func TestCtx_Root(t *testing.T) {
	t.Parallel()
	ctx := gofuncy.Ctx(t.Context()).Root()
	assert.Equal(t, gofuncy.NameRoot, gofuncy.Ctx(ctx).Name())
}

func TestCtx_Parent(t *testing.T) {
	t.Parallel()
	assert.Empty(t, gofuncy.Ctx(t.Context()).Parent())
}

func TestContext_nestedGoRoutines(t *testing.T) {
	type result struct {
		name   string
		parent string
	}

	ch := make(chan result, 1)
	done := make(chan struct{})

	gofuncy.Go(t.Context(), "parent",
		func(ctx context.Context) error {
			gofuncy.Go(ctx, "child",
				func(ctx context.Context) error {
					ch <- result{
						name:   gofuncy.NameFromContext(ctx),
						parent: gofuncy.ParentFromContext(ctx),
					}

					close(done)

					return nil
				},
			)

			<-done

			return nil
		},
	)

	select {
	case r := <-ch:
		assert.Equal(t, "child", r.name)
		assert.Equal(t, "parent", r.parent)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for nested Go routines")
	}
}

func TestContext_existingDeadlineShorterThanTimeout(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	// Parent context has a 20ms deadline
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	gofuncy.Go(ctx, "deadline-test",
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		// WithTimeout sets a 5s timeout, but parent's 20ms deadline wins
		gofuncy.WithTimeout(5*time.Second),
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("timed out — parent deadline should have triggered within 20ms")
	}
}
