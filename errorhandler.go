package gofuncy

import (
	"context"
)

// ErrorHandler is a callback for handling errors from fire-and-forget goroutines.
type ErrorHandler func(ctx context.Context, err error)
