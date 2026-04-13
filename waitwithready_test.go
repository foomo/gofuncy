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

func ExampleWaitWithReady() {
	proceed := make(chan struct{})

	wait := gofuncy.WaitWithReady(context.Background(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		fmt.Println("initializing")
		ready()

		<-proceed
		fmt.Println("done")

		return nil
	})

	fmt.Println("ready")
	close(proceed)

	if err := wait(); err != nil {
		fmt.Println("error:", err)
	}
	// Output:
	// initializing
	// ready
	// done
}

func TestWaitWithReady_basic(t *testing.T) {
	t.Parallel()

	var readyCalled atomic.Bool

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		readyCalled.Store(true)
		ready()
		time.Sleep(20 * time.Millisecond)

		return nil
	})

	assert.True(t, readyCalled.Load())
	require.NoError(t, wait())
}

func TestWaitWithReady_noReadyCall(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		return fmt.Errorf("init failed")
	})

	require.EqualError(t, wait(), "init failed")
}

func TestWaitWithReady_panicRecovery(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		panic("before ready")
	})

	err := wait()

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "before ready", panicErr.Value)
}

func TestWaitWithReady_multipleReadyCalls(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		ready()
		ready()
		ready()

		return nil
	})

	require.NoError(t, wait())
}

func TestWaitWithReady_multipleWaitCalls(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		ready()

		return fmt.Errorf("fail")
	})

	err1 := wait()
	err2 := wait()

	require.EqualError(t, err1, "fail")
	assert.Equal(t, err1, err2)
}

func TestWaitWithReady_withTimeout(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithReady(t.Context(), func(ctx context.Context, ready gofuncy.ReadyFunc) error {
		ready()
		<-ctx.Done()

		return ctx.Err()
	}, gofuncy.WithTimeout(10*time.Millisecond))

	require.ErrorIs(t, wait(), context.DeadlineExceeded)
}
