package gofuncy

import (
	"context"
)

type Func func(ctx context.Context) error

// Middleware wraps a Func to add cross-cutting behavior.
type Middleware func(Func) Func
