package gofuncy

import (
	"context"
	"errors"
)

// FallbackOption configures fallback behavior.
type FallbackOption func(*fallbackConfig)

type fallbackConfig struct {
	fallbackIf func(error) bool
}

// Fallback returns a Middleware that calls fn when the wrapped function returns
// an error, allowing graceful degradation. The fallback function receives the
// original context and error, and may return nil to suppress the error or a
// different error. By default, context errors and panics bypass the fallback;
// use FallbackIf to customize which errors trigger it.
func Fallback(fn func(ctx context.Context, err error) error, opts ...FallbackOption) Middleware {
	cfg := fallbackConfig{
		fallbackIf: defaultFallbackIf,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next Func) Func {
		return func(ctx context.Context) error {
			err := next(ctx)
			if err == nil {
				return nil
			}

			if !cfg.fallbackIf(err) {
				return err
			}

			return fn(ctx, err)
		}
	}
}

// FallbackIf sets a custom function to determine whether an error should
// trigger the fallback. By default, all errors except context errors and
// panics trigger the fallback.
func FallbackIf(fn func(error) bool) FallbackOption {
	return func(c *fallbackConfig) {
		c.fallbackIf = fn
	}
}

func defaultFallbackIf(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var panicErr *PanicError

	return !errors.As(err, &panicErr)
}
