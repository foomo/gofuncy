package gofuncy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/require"
)

func ExampleGoBackground() {
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — goroutine still runs

	gofuncy.GoBackground(ctx, func(ctx context.Context) error {
		defer close(done)

		fmt.Println("running despite canceled context")

		return nil
	})

	<-done
	// Output:
	// running despite canceled context
}

func TestGoBackground_basic(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	gofuncy.GoBackground(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for GoBackground to complete")
	}
}

func TestGoBackground_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})

	gofuncy.GoBackground(ctx,
		func(ctx context.Context) error {
			close(done)
			return nil
		},
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out: goroutine should run despite canceled context")
	}
}

func TestAsyncBackground_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := gofuncy.AsyncBackground(ctx,
		func(ctx context.Context) error {
			return nil
		},
	)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out: goroutine should run despite canceled context")
	}
}
