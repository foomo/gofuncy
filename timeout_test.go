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

func TestTimeout_completesBeforeDeadline(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithTimeout(time.Second))

	require.NoError(t, g.Wait())
}

func TestTimeout_exceedsDeadline(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, gofuncy.WithTimeout(10*time.Millisecond))

	err := g.Wait()
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestTimeout_eachAttemptGetsOwnDeadline(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			// First two attempts: wait for timeout
			<-ctx.Done()
			return ctx.Err()
		}
		// Third attempt: succeed immediately
		return nil
	},
		gofuncy.WithTimeout(20*time.Millisecond),
		gofuncy.WithRetry(3,
			gofuncy.RetryBackoff(gofuncy.BackoffConstant(0)),
			gofuncy.RetryIf(func(err error) bool { return true }),
		),
	)

	require.NoError(t, g.Wait())
	assert.Equal(t, int32(3), calls.Load())
}

func TestTimeout_doesNotAffectSuccessfulCalls(t *testing.T) {
	t.Parallel()

	start := time.Now()

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		return nil
	}, gofuncy.WithTimeout(time.Second))

	require.NoError(t, g.Wait())
	assert.Less(t, time.Since(start), 100*time.Millisecond)
}

func TestTimeout_propagatesNonTimeoutError(t *testing.T) {
	t.Parallel()

	g := gofuncy.NewGroup(t.Context())
	g.Add(func(ctx context.Context) error {
		return fmt.Errorf("application error")
	}, gofuncy.WithTimeout(time.Second))

	err := g.Wait()
	require.EqualError(t, err, "application error")
}
