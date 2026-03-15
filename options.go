package gofuncy

import (
	"context"
	"log/slog"
	"sync"
)

// ErrorHandler is a callback for handling errors from fire-and-forget goroutines.
type ErrorHandler func(ctx context.Context, err error)

type (
	options struct {
		l    *slog.Logger
		name string
		// telemetry
		tracing        bool
		upDownMetric   bool
		counterMetric  bool
		durationMetric bool
		// concurrency control
		limit    int
		failFast bool
		// error handling
		errorHandler ErrorHandler
	}
	Option func(*options)
)

var optionsPool = sync.Pool{New: func() any { return &options{} }}

func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(o *options) {
		o.l = l
	}
}

func WithTracing() Option {
	return func(o *options) {
		o.tracing = true
	}
}

func WithUpDownMetric() Option {
	return func(o *options) {
		o.upDownMetric = true
	}
}

func WithDurationMetric() Option {
	return func(o *options) {
		o.durationMetric = true
	}
}

func WithCounterMetric() Option {
	return func(o *options) {
		o.counterMetric = true
	}
}

// WithLimit sets the maximum number of concurrent goroutines for Group and Map.
func WithLimit(n int) Option {
	return func(o *options) {
		o.limit = n
	}
}

// WithFailFast cancels remaining goroutines on first error in Group and Map.
func WithFailFast() Option {
	return func(o *options) {
		o.failFast = true
	}
}

// WithErrorHandler sets a custom error handler for fire-and-forget Go.
func WithErrorHandler(h ErrorHandler) Option {
	return func(o *options) {
		o.errorHandler = h
	}
}

// reset clears all fields in options for reuse from pool.
func (o *options) reset() {
	o.l = nil
	o.name = NameNoName
	o.tracing = false
	o.upDownMetric = false
	o.counterMetric = false
	o.durationMetric = false
	o.limit = 0
	o.failFast = false
	o.errorHandler = nil
}

func getOptions(opts []Option) *options {
	var o *options
	if opt, ok := optionsPool.Get().(*options); ok {
		o = opt
	} else {
		o = &options{}
	}

	o.reset()

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}
