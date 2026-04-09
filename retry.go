package gofuncy

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

// Backoff returns the delay before the nth retry attempt (0-indexed).
type Backoff func(attempt int) time.Duration

// RetryOption configures retry behavior.
type RetryOption func(*retryConfig)

type retryConfig struct {
	backoff func(attempt int) time.Duration
	retryIf func(error) bool
	onRetry func(ctx context.Context, attempt int, err error)
	meter   metric.Meter
	name    string
}

// Retry returns a Middleware that retries the wrapped function up to
// maxAttempts times total (1 = no retry, 3 = initial + up to 2 retries).
func Retry(maxAttempts int, opts ...RetryOption) Middleware {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	cfg := retryConfig{
		backoff: BackoffExponential(100*time.Millisecond, 2, 30*time.Second),
		retryIf: defaultRetryIf,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	retries, err := gofuncyconv.NewGoroutinesRetries(cfg.meter)
	if err != nil {
		otel.Handle(err)
	}

	return func(fn Func) Func {
		return func(ctx context.Context) error {
			var err error
			for attempt := 0; attempt < maxAttempts; attempt++ {
				err = fn(ctx)
				if err == nil {
					return nil
				}

				if !cfg.retryIf(err) {
					return err
				}

				if attempt == maxAttempts-1 {
					break
				}

				retries.Add(ctx, 1, cfg.name)

				if cfg.onRetry != nil {
					cfg.onRetry(ctx, attempt+1, err)
				}

				delay := cfg.backoff(attempt)

				t := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					t.Stop()
					return ctx.Err()
				case <-t.C:
				}
			}

			return err
		}
	}
}

// RetryBackoff sets a custom backoff strategy.
func RetryBackoff(b Backoff) RetryOption {
	return func(c *retryConfig) {
		c.backoff = b
	}
}

// RetryIf sets a custom function to determine whether an error is retryable.
func RetryIf(fn func(error) bool) RetryOption {
	return func(c *retryConfig) {
		c.retryIf = fn
	}
}

// RetryOnRetry sets a callback invoked before each retry attempt.
// The attempt parameter is 1-indexed (1 = first retry).
func RetryOnRetry(fn func(ctx context.Context, attempt int, err error)) RetryOption {
	return func(c *retryConfig) {
		c.onRetry = fn
	}
}

// BackoffConstant returns a Backoff that always waits the same duration.
func BackoffConstant(d time.Duration) Backoff {
	return func(_ int) time.Duration {
		return d
	}
}

// BackoffExponential returns a Backoff with exponential growth, jitter, and a cap.
// The delay for attempt n is: min(initial * multiplier^n, max) +/- 25% jitter.
func BackoffExponential(initial time.Duration, multiplier float64, maxDelay time.Duration) Backoff {
	return func(attempt int) time.Duration {
		delay := float64(initial) * math.Pow(multiplier, float64(attempt))
		if delay > float64(maxDelay) {
			delay = float64(maxDelay)
		}

		jitter := delay * 0.25
		delay = delay - jitter + rand.Float64()*2*jitter //nolint:gosec

		return time.Duration(delay)
	}
}

func retryWithMeter(m metric.Meter, name string) RetryOption {
	return func(c *retryConfig) {
		c.meter = m
		c.name = name
	}
}

func defaultRetryIf(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var panicErr *PanicError

	return !errors.As(err, &panicErr)
}
