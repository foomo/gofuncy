package gofuncy

import (
	"context"
)

// Map transforms items concurrently, preserving input order.
// Returns results and the joined errors.
// All GroupOption options apply (WithLimit, WithFailFast, telemetry, etc.).
// Use WithName to set a custom metric/tracing label; defaults to "gofuncy.map".
func Map[T, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error), opts ...GroupOption) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]R, len(items))

	g := NewGroup(ctx, opts...)

	for i, item := range items {
		g.Add(func(ctx context.Context) error {
			r, err := fn(ctx, item)
			if err != nil {
				return err
			}

			results[i] = r

			return nil
		})
	}

	return results, g.Wait() //nolint:contextcheck
}
