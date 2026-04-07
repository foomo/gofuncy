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
	GoOptions | AsyncOptions | GroupOptions | MapOptions
}

type hasConcurrentOptions interface {
	GroupOptions | MapOptions
}

// ------------------------------------------------------------------------------------------------
// ~ Shared options (all operation types)
// ------------------------------------------------------------------------------------------------

func WithName[T hasBaseOptions](name string) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.name = name
		case *AsyncOptions:
			x.name = name
		case *GroupOptions:
			x.name = name
		case *MapOptions:
			x.name = name
		}
	}
}

func WithLogger[T hasBaseOptions](l *slog.Logger) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.l = l
		case *AsyncOptions:
			x.l = l
		case *GroupOptions:
			x.l = l
		case *MapOptions:
			x.l = l
		}
	}
}

func WithTracing[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.tracing = true
		case *AsyncOptions:
			x.tracing = true
		case *GroupOptions:
			x.tracing = true
		case *MapOptions:
			x.tracing = true
		}
	}
}

func WithUpDownMetric[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.upDownMetric = true
		case *AsyncOptions:
			x.upDownMetric = true
		case *GroupOptions:
			x.upDownMetric = true
		case *MapOptions:
			x.upDownMetric = true
		}
	}
}

func WithDurationMetric[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.durationMetric = true
		case *AsyncOptions:
			x.durationMetric = true
		case *GroupOptions:
			x.durationMetric = true
		case *MapOptions:
			x.durationMetric = true
		}
	}
}

func WithCounterMetric[T hasBaseOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.counterMetric = true
		case *AsyncOptions:
			x.counterMetric = true
		case *GroupOptions:
			x.counterMetric = true
		case *MapOptions:
			x.counterMetric = true
		}
	}
}

func WithMeterProvider[T hasBaseOptions](mp metric.MeterProvider) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.meterProvider = mp
		case *AsyncOptions:
			x.meterProvider = mp
		case *GroupOptions:
			x.meterProvider = mp
		case *MapOptions:
			x.meterProvider = mp
		}
	}
}

func WithTracerProvider[T hasBaseOptions](tp trace.TracerProvider) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GoOptions:
			x.tracerProvider = tp
		case *AsyncOptions:
			x.tracerProvider = tp
		case *GroupOptions:
			x.tracerProvider = tp
		case *MapOptions:
			x.tracerProvider = tp
		}
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Go-only options
// ------------------------------------------------------------------------------------------------

func WithTimeout[T GoOptions](timeout time.Duration) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.timeout = timeout
		}
	}
}

func WithCallerSkip[T GoOptions](skip int) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.callerSkip = skip
		}
	}
}

func WithErrorHandler[T GoOptions](h ErrorHandler) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) { //nolint:gocritic // singleCaseSwitch
		case *GoOptions:
			x.errorHandler = h
		}
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Concurrent options (Group and Map)
// ------------------------------------------------------------------------------------------------

func WithLimit[T hasConcurrentOptions](n int) Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GroupOptions:
			x.limit = n
		case *MapOptions:
			x.limit = n
		}
	}
}

func WithFailFast[T hasConcurrentOptions]() Option[T] {
	return func(o *T) {
		switch x := any(o).(type) {
		case *GroupOptions:
			x.failFast = true
		case *MapOptions:
			x.failFast = true
		}
	}
}
