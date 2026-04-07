package gofuncy

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Option is a generic functional option for configuring gofuncy operations.
type Option[T any] func(*T)

type hasBaseOptions interface {
	GoOptions
}

// ------------------------------------------------------------------------------------------------
// ~ Shared options (all operation types)
// ------------------------------------------------------------------------------------------------

// WithName sets a custom name for the routine and updates the corresponding GoOptions field.
func WithName[T hasBaseOptions](name string) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.name = name
		}
	}
}

// WithLogger configures the logger for the operation using the provided *slog.Logger.
func WithLogger[T hasBaseOptions](l *slog.Logger) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.l = l
		}
	}
}

// WithTracing enables tracing for the operation by setting the tracing flag in the GoOptions.
func WithTracing[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.tracing = true
		}
	}
}

// WithStartedCounter enables the started counter for the operation by setting the startedCounter flag in the GoOptions.
func WithStartedCounter[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.startedCounter = true
		}
	}
}

// WithFinishedCounter enables the finished counter for the operation by setting the finishedCounter flag in the GoOptions.
func WithFinishedCounter[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.finishedCounter = true
		}
	}
}

// WithErrorCounter enables the error counter for the operation by setting the errorCounter flag in the GoOptions.
func WithErrorCounter[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.errorCounter = true
		}
	}
}

// WithActiveUpDownCounter enables the active up-down counter for the operation by setting the activeUpDownCounter flag in the GoOptions.
func WithActiveUpDownCounter[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.activeUpDownCounter = true
		}
	}
}

// WithDurationHistogram enables the duration histogram for the operation by setting the durationHistogram flag in the GoOptions.
func WithDurationHistogram[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.durationHistogram = true
		}
	}
}

// WithDurationTimer enables the duration timer for the operation by setting the durationTimer flag in the GoOptions.
func WithMiddleware[T hasBaseOptions](m ...Middleware) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.middlewares = append(x.middlewares, m...)
		}
	}
}

// WithMeterProvider enables the meter provider for the operation by setting the meterProvider field in the GoOptions.
func WithMeterProvider[T hasBaseOptions](mp metric.MeterProvider) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.meterProvider = mp
		}
	}
}

// WithTracerProvider enables the tracer provider for the operation by setting the tracerProvider field in the GoOptions.
func WithTracerProvider[T hasBaseOptions](tp trace.TracerProvider) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.tracerProvider = tp
		}
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Go-only options
// ------------------------------------------------------------------------------------------------

// WithTimeout enables the timeout for the operation by setting the timeout field in the GoOptions.
func WithTimeout[T GoOptions](timeout time.Duration) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.timeout = timeout
		}
	}
}

// WithCallerSkip enables the caller skip for the operation by setting the callerSkip field in the GoOptions.
func WithCallerSkip[T GoOptions](skip int) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.callerSkip = skip
		}
	}
}

// WithErrorHandler enables the error handler for the operation by setting the errorHandler field in the GoOptions.
func WithErrorHandler[T GoOptions](h ErrorHandler) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.errorHandler = h
		}
	}
}
