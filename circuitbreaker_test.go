package gofuncy_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_passesWhenClosed(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(gofuncy.CircuitBreakerThreshold(3))

	g := gofuncy.NewGroup(t.Context(), "cb-closed")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))

	require.NoError(t, g.Wait())
}

func TestCircuitBreaker_opensAfterThreshold(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(3),
		gofuncy.CircuitBreakerCooldown(time.Hour),
	)

	// Trip the circuit with 3 failures
	for i := range 3 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-trip-%d", i))
		g.Add("task", func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// Next call should be rejected
	g := gofuncy.NewGroup(t.Context(), "cb-open")
	g.Add("task", func(ctx context.Context) error {
		t.Fatal("should not be called")
		return nil
	}, gofuncy.WithCircuitBreaker(cb))

	err := g.Wait()
	require.ErrorIs(t, err, gofuncy.ErrCircuitOpen)
}

func TestCircuitBreaker_resetsOnSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	cb := gofuncy.NewCircuitBreaker(gofuncy.CircuitBreakerThreshold(3))

	// 2 failures (under threshold)
	for i := range 2 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-fail-%d", i))
		g.Add("task", func(ctx context.Context) error {
			calls.Add(1)
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// 1 success — resets counter
	g := gofuncy.NewGroup(t.Context(), "cb-success")
	g.Add("task", func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())

	// 2 more failures — still under threshold (counter was reset)
	for i := range 2 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-fail2-%d", i))
		g.Add("task", func(ctx context.Context) error {
			calls.Add(1)
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// Should still pass through (2 < 3)
	g = gofuncy.NewGroup(t.Context(), "cb-still-closed")
	g.Add("task", func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())
}

func TestCircuitBreaker_halfOpenProbeSucceeds(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(2),
		gofuncy.CircuitBreakerCooldown(10*time.Millisecond),
	)

	// Trip the circuit
	for i := range 2 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-trip-%d", i))
		g.Add("task", func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Probe should succeed and close the circuit
	g := gofuncy.NewGroup(t.Context(), "cb-probe")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())

	// Circuit should be closed — normal calls work
	g = gofuncy.NewGroup(t.Context(), "cb-recovered")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())
}

func TestCircuitBreaker_halfOpenProbeFails(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(2),
		gofuncy.CircuitBreakerCooldown(10*time.Millisecond),
	)

	// Trip the circuit
	for i := range 2 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-trip-%d", i))
		g.Add("task", func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Probe fails — circuit should reopen
	g := gofuncy.NewGroup(t.Context(), "cb-probe-fail")
	g.Add("task", func(ctx context.Context) error {
		return fmt.Errorf("still broken")
	}, gofuncy.WithCircuitBreaker(cb))
	err := g.Wait()
	require.EqualError(t, err, "still broken")

	// Next call should be rejected immediately
	g = gofuncy.NewGroup(t.Context(), "cb-reopened")
	g.Add("task", func(ctx context.Context) error {
		t.Fatal("should not be called")
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	err = g.Wait()
	require.ErrorIs(t, err, gofuncy.ErrCircuitOpen)
}

func TestCircuitBreaker_ignoresContextErrors(t *testing.T) {
	t.Parallel()

	cb := gofuncy.NewCircuitBreaker(gofuncy.CircuitBreakerThreshold(1))

	g := gofuncy.NewGroup(t.Context(), "cb-ctx-err")
	g.Add("task", func(ctx context.Context) error {
		return context.Canceled
	}, gofuncy.WithCircuitBreaker(cb))
	_ = g.Wait()

	// Circuit should still be closed
	g = gofuncy.NewGroup(t.Context(), "cb-after-ctx")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())
}

func TestCircuitBreaker_customFailureIf(t *testing.T) {
	t.Parallel()

	retryable := fmt.Errorf("retryable")

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(1),
		gofuncy.CircuitBreakerIf(func(err error) bool {
			return !errors.Is(err, retryable)
		}),
	)

	// This error should not count as a failure
	g := gofuncy.NewGroup(t.Context(), "cb-custom-skip")
	g.Add("task", func(ctx context.Context) error {
		return retryable
	}, gofuncy.WithCircuitBreaker(cb))
	_ = g.Wait()

	// Circuit should still be closed
	g = gofuncy.NewGroup(t.Context(), "cb-custom-pass")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())
}

func TestCircuitBreaker_onStateChange(t *testing.T) {
	t.Parallel()

	var transitions []string

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(2),
		gofuncy.CircuitBreakerCooldown(10*time.Millisecond),
		gofuncy.CircuitBreakerOnStateChange(func(from, to gofuncy.CircuitState) {
			transitions = append(transitions, fmt.Sprintf("%s->%s", from, to))
		}),
	)

	// Trip the circuit: closed -> open
	for i := range 2 {
		g := gofuncy.NewGroup(t.Context(), fmt.Sprintf("cb-state-%d", i))
		g.Add("task", func(ctx context.Context) error {
			return fmt.Errorf("fail")
		}, gofuncy.WithCircuitBreaker(cb))
		_ = g.Wait()
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Probe succeeds: open -> half-open -> closed
	g := gofuncy.NewGroup(t.Context(), "cb-state-recover")
	g.Add("task", func(ctx context.Context) error {
		return nil
	}, gofuncy.WithCircuitBreaker(cb))
	require.NoError(t, g.Wait())

	assert.Equal(t, []string{
		"closed->open",
		"open->half-open",
		"half-open->closed",
	}, transitions)
}

func TestCircuitBreaker_composesWithRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	cb := gofuncy.NewCircuitBreaker(
		gofuncy.CircuitBreakerThreshold(5),
		gofuncy.CircuitBreakerCooldown(time.Hour),
	)

	// Circuit breaker wraps outside retry
	g := gofuncy.NewGroup(t.Context(), "cb-retry")
	g.Add("task", func(ctx context.Context) error {
		calls.Add(1)
		return fmt.Errorf("fail")
	},
		gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(0))),
		gofuncy.WithCircuitBreaker(cb),
	)

	err := g.Wait()
	require.Error(t, err)
	// Retry exhausted 3 attempts, circuit sees 1 final failure
	assert.Equal(t, int32(3), calls.Load())
}
