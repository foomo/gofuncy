package gofuncy

import (
	"context"
)

// Func represents a function that can be executed within a routine context.
type Func func(ctx context.Context) error

// Middleware wraps a Func to add cross-cutting behavior.
type Middleware func(Func) Func

// ReadyFunc signals that a goroutine has completed initialization.
// Safe to call multiple times.
type ReadyFunc func()

// StopFunc cancels a goroutine's context, signaling it to shut down.
// Safe to call multiple times.
type StopFunc func()
