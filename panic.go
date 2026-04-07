package gofuncy

import (
	"fmt"
	"runtime/debug"
)

// PanicError wraps a recovered panic value with its stack trace.
type PanicError struct {
	Value any
	Stack []byte
}

// Error implements the error interface for PanicError.
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}

// recoverError converts a panic into a *PanicError assigned to the given error pointer.
// Usage: defer recoverError(&err)
func recoverError(err *error) {
	if r := recover(); r != nil {
		*err = &PanicError{
			Value: r,
			Stack: debug.Stack(),
		}
	}
}
