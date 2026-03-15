package gofuncy

import (
	"context"
	"errors"
)

// Map transforms input items concurrently, preserving order.
// Returns results and errors.Join of all failures.
// Use WithFailFast to cancel on first error.
// Use WithLimit(n) to bound concurrent goroutines.
func Map[T, R any](ctx context.Context, input []T, fn func(ctx context.Context, v T) (R, error), opts ...Option) ([]R, error) {
	if len(input) == 0 {
		return nil, nil
	}

	results := make([]R, len(input))

	// wrap each input item into a Func that writes to results[i]
	fns := make([]Func, len(input))
	for i, v := range input {
		fns[i] = func(ctx context.Context) error {
			r, err := fn(ctx, v)
			if err != nil {
				return err
			}

			results[i] = r

			return nil
		}
	}

	o := getOptions(opts)
	defer optionsPool.Put(o)

	errs := run(ctx, fns, o)

	return results, errors.Join(errs...)
}

// MapBackground is like Map but detaches from the parent context's cancellation.
// The goroutines will continue running even if the parent context is canceled.
func MapBackground[T, R any](ctx context.Context, input []T, fn func(ctx context.Context, v T) (R, error), opts ...Option) ([]R, error) {
	return Map(context.WithoutCancel(ctx), input, fn, opts...)
}
