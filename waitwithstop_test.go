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

func ExampleWaitWithStop() {
	wait := gofuncy.WaitWithStop(context.Background(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		fmt.Println("started")
		stop()
		<-ctx.Done()
		fmt.Println("stopped")

		return nil
	})

	err := wait()
	fmt.Println("error:", err)
	// Output:
	// started
	// stopped
	// error: <nil>
}

func TestWaitWithStop_basic(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		stop()
		<-ctx.Done()

		return nil
	})

	require.NoError(t, wait())
}

func TestWaitWithStop_returnsError(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		return fmt.Errorf("boom")
	})

	require.EqualError(t, wait(), "boom")
}

func TestWaitWithStop_panicRecovery(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		panic("oops")
	})

	err := wait()

	var panicErr *gofuncy.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "oops", panicErr.Value)
}

func TestWaitWithStop_stopCancelsContext(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		stop()

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			return fmt.Errorf("context was not canceled")
		}
	})

	require.NoError(t, wait())
}

func TestWaitWithStop_multipleWaitCalls(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		return fmt.Errorf("fail")
	})

	err1 := wait()
	err2 := wait()

	require.EqualError(t, err1, "fail")
	assert.Equal(t, err1, err2)
}

func TestWaitWithStop_withTimeout(t *testing.T) {
	t.Parallel()

	wait := gofuncy.WaitWithStop(t.Context(), func(ctx context.Context, stop gofuncy.StopFunc) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(10*time.Millisecond))

	require.ErrorIs(t, wait(), context.DeadlineExceeded)
}
