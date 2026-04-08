package gofuncy

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/semaphore"
)

// GoOption configures Go() and Group.Add() calls.
type GoOption interface {
	applyGo(o *options)
}

// GroupOption configures NewGroup() calls.
type GroupOption interface {
	applyGroup(o *options)
}

// ------------------------------------------------------------------------------------------------
// ~ Carrier types
// ------------------------------------------------------------------------------------------------

// baseOpt implements both GoOption and GroupOption.
type baseOpt func(*options)

func (f baseOpt) applyGo(o *options)    { f(o) }
func (f baseOpt) applyGroup(o *options) { f(o) }

// goOnlyOpt implements only GoOption.
type goOnlyOpt func(*options)

func (f goOnlyOpt) applyGo(o *options) { f(o) }

// groupOnlyOpt implements only GroupOption.
type groupOnlyOpt func(*options)

func (f groupOnlyOpt) applyGroup(o *options) { f(o) }

// ------------------------------------------------------------------------------------------------
// ~ Options struct
// ------------------------------------------------------------------------------------------------

type options struct {
	l            *slog.Logger
	name         string
	timeout      time.Duration
	callerSkip   int
	errorHandler ErrorHandler
	// telemetry
	tracing             bool
	startedCounter      bool
	finishedCounter     bool
	errorCounter        bool
	activeUpDownCounter bool
	durationHistogram   bool
	// middleware
	middlewares []Middleware
	// telemetry providers
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	// concurrency
	limiter *semaphore.Weighted
	// group-specific
	limit    int
	failFast bool
}

func (o options) meter() metric.Meter {
	mp := o.meterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	return mp.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL))
}

func (o options) tracer() trace.Tracer {
	tp := o.tracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(ScopeName)
}

// merge returns a new options with override values applied on top of o.
// Booleans are OR'd, slices are appended, pointers/interfaces use override if non-zero.
// Group-specific fields (limit, failFast) are not merged.
func (o options) merge(override options) options {
	if override.name != "" && override.name != NameNoName {
		o.name = override.name
	}

	if override.l != nil {
		o.l = override.l
	}

	if len(override.middlewares) > 0 {
		merged := make([]Middleware, len(o.middlewares)+len(override.middlewares))
		copy(merged, o.middlewares)
		copy(merged[len(o.middlewares):], override.middlewares)
		o.middlewares = merged
	}

	if override.timeout > 0 {
		o.timeout = override.timeout
	}

	o.tracing = o.tracing || override.tracing
	o.startedCounter = o.startedCounter || override.startedCounter
	o.finishedCounter = o.finishedCounter || override.finishedCounter
	o.errorCounter = o.errorCounter || override.errorCounter
	o.activeUpDownCounter = o.activeUpDownCounter || override.activeUpDownCounter
	o.durationHistogram = o.durationHistogram || override.durationHistogram

	if override.meterProvider != nil {
		o.meterProvider = override.meterProvider
	}

	if override.tracerProvider != nil {
		o.tracerProvider = override.tracerProvider
	}

	if override.limiter != nil {
		o.limiter = override.limiter
	}

	return o
}

// ------------------------------------------------------------------------------------------------
// ~ Constructors
// ------------------------------------------------------------------------------------------------

func newGoOptions(opts []GoOption) options {
	o := options{
		name: NameNoName,
	}

	for _, opt := range opts {
		if opt != nil {
			opt.applyGo(&o)
		}
	}

	return o
}

func newGroupOptions(opts []GroupOption) options {
	o := options{
		name: NameNoName,
	}

	for _, opt := range opts {
		if opt != nil {
			opt.applyGroup(&o)
		}
	}

	return o
}

// newGoOverrideOptions creates options without default name, used for Group.Add() merging.
func newGoOverrideOptions(opts []GoOption) options {
	var o options

	for _, opt := range opts {
		if opt != nil {
			opt.applyGo(&o)
		}
	}

	return o
}

// ------------------------------------------------------------------------------------------------
// ~ Shared options (Go, NewGroup, Add)
// ------------------------------------------------------------------------------------------------

// WithName sets a custom name for the routine.
func WithName(name string) baseOpt {
	return func(o *options) {
		o.name = name
	}
}

// WithLogger configures the logger for the operation.
func WithLogger(l *slog.Logger) baseOpt {
	return func(o *options) {
		o.l = l
	}
}

// WithTracing enables tracing for the operation.
func WithTracing() baseOpt {
	return func(o *options) {
		o.tracing = true
	}
}

// WithStartedCounter enables the started counter metric.
func WithStartedCounter() baseOpt {
	return func(o *options) {
		o.startedCounter = true
	}
}

// WithFinishedCounter enables the finished counter metric.
func WithFinishedCounter() baseOpt {
	return func(o *options) {
		o.finishedCounter = true
	}
}

// WithErrorCounter enables the error counter metric.
func WithErrorCounter() baseOpt {
	return func(o *options) {
		o.errorCounter = true
	}
}

// WithActiveUpDownCounter enables the active up-down counter metric.
func WithActiveUpDownCounter() baseOpt {
	return func(o *options) {
		o.activeUpDownCounter = true
	}
}

// WithDurationHistogram enables the duration histogram metric.
func WithDurationHistogram() baseOpt {
	return func(o *options) {
		o.durationHistogram = true
	}
}

// WithMiddleware appends middleware to the operation's middleware chain.
func WithMiddleware(m ...Middleware) baseOpt {
	return func(o *options) {
		o.middlewares = append(o.middlewares, m...)
	}
}

// WithMeterProvider sets a custom meter provider.
func WithMeterProvider(mp metric.MeterProvider) baseOpt {
	return func(o *options) {
		o.meterProvider = mp
	}
}

// WithTracerProvider sets a custom tracer provider.
func WithTracerProvider(tp trace.TracerProvider) baseOpt {
	return func(o *options) {
		o.tracerProvider = tp
	}
}

// WithLimiter sets a shared weighted semaphore for concurrency control.
func WithLimiter(l *semaphore.Weighted) baseOpt {
	return func(o *options) {
		o.limiter = l
	}
}

// WithTimeout sets a timeout for the operation.
func WithTimeout(timeout time.Duration) baseOpt {
	return func(o *options) {
		o.timeout = timeout
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Go-only options (Go, Add)
// ------------------------------------------------------------------------------------------------

// WithCallerSkip sets the caller skip for error reporting.
func WithCallerSkip(skip int) goOnlyOpt {
	return func(o *options) {
		o.callerSkip = skip
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(h ErrorHandler) goOnlyOpt {
	return func(o *options) {
		o.errorHandler = h
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Group-only options (NewGroup)
// ------------------------------------------------------------------------------------------------

// WithLimit sets the maximum number of concurrently executing functions in a Group.
func WithLimit(n int) groupOnlyOpt {
	return func(o *options) {
		o.limit = n
	}
}

// WithFailFast configures the Group to cancel remaining functions on first error.
func WithFailFast() groupOnlyOpt {
	return func(o *options) {
		o.failFast = true
	}
}
