package gofuncy

import (
	"context"
)

// Func represents a function that can be executed within a routine context.
type Func func(ctx context.Context) error

// Middleware wraps a Func to add cross-cutting behavior.
type Middleware func(Func) Func
