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

// Unwrap returns the underlying error if the panic value implements error.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}

	return nil
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
