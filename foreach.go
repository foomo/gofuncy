package gofuncy

import (
	"context"
)

// All executes fn for each item concurrently and returns the joined errors.
// All GroupOption options apply (WithLimit, WithFailFast, telemetry, etc.).
func All[T any](ctx context.Context, name string, items []T, fn func(ctx context.Context, item T) error, opts ...GroupOption) error {
	if len(items) == 0 {
		return nil
	}

	g := NewGroup(ctx, name, opts...)

	for _, item := range items {
		g.Add(name, func(ctx context.Context) error {
			return fn(ctx, item)
		})
	}

	return g.Wait() //nolint:contextcheck
}
