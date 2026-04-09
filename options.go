package gofuncy

import (
	"context"
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

type options struct { //nolint:recvcheck
	l              *slog.Logger
	name           string
	timeout        time.Duration
	callerSkip     int
	errorHandler   ErrorHandler
	stallThreshold time.Duration
	stallHandler   StallHandler
	// telemetry
	tracing             bool
	startedCounter      bool
	errorCounter        bool
	activeUpDownCounter bool
	durationHistogram   bool
	// resilience
	retryAttempts  int
	retryOpts      []RetryOption
	circuitBreaker *CircuitBreaker
	fallbackFn     func(context.Context, error) error
	fallbackOpts   []FallbackOption
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

// meter returns the OTel Meter for this scope. The OTel SDK caches both Meter
// instances and instrument handles internally, so repeated calls are cheap.
func (o *options) meter() metric.Meter {
	mp := o.meterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	return mp.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL))
}

// tracer returns the OTel Tracer for this scope. The OTel SDK caches Tracer
// instances internally, so repeated calls are cheap.
func (o *options) tracer() trace.Tracer {
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

	if override.stallThreshold > 0 {
		o.stallThreshold = override.stallThreshold
	}

	if override.stallHandler != nil {
		o.stallHandler = override.stallHandler
	}

	o.tracing = o.tracing || override.tracing
	o.startedCounter = o.startedCounter || override.startedCounter
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

	if override.retryAttempts > 0 {
		o.retryAttempts = override.retryAttempts
		o.retryOpts = override.retryOpts
	}

	if override.circuitBreaker != nil {
		o.circuitBreaker = override.circuitBreaker
	}

	if override.fallbackFn != nil {
		o.fallbackFn = override.fallbackFn
		o.fallbackOpts = override.fallbackOpts
	}

	return o
}

// ------------------------------------------------------------------------------------------------
// ~ Constructors
// ------------------------------------------------------------------------------------------------

func newGoOptions(opts []GoOption) options {
	o := options{
		tracing:             true,
		startedCounter:      true,
		errorCounter:        true,
		activeUpDownCounter: true,
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
		tracing:             true,
		startedCounter:      true,
		errorCounter:        true,
		activeUpDownCounter: true,
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

// WithLogger configures the logger for the operation.
func WithLogger(l *slog.Logger) baseOpt {
	return func(o *options) {
		o.l = l
	}
}

// WithDurationHistogram enables the duration histogram metric.
func WithDurationHistogram() baseOpt {
	return func(o *options) {
		o.durationHistogram = true
	}
}

// WithoutTracing disables tracing for the operation.
func WithoutTracing() baseOpt {
	return func(o *options) {
		o.tracing = false
	}
}

// WithoutStartedCounter disables the started counter metric.
func WithoutStartedCounter() baseOpt {
	return func(o *options) {
		o.startedCounter = false
	}
}

// WithoutErrorCounter disables the error counter metric.
func WithoutErrorCounter() baseOpt {
	return func(o *options) {
		o.errorCounter = false
	}
}

// WithoutActiveUpDownCounter disables the active up-down counter metric.
func WithoutActiveUpDownCounter() baseOpt {
	return func(o *options) {
		o.activeUpDownCounter = false
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

// WithStallThreshold enables stall detection. If a goroutine runs longer than
// the threshold, a warning is logged and a metric is emitted. The goroutine is
// not cancelled. Use WithStallHandler to customize the callback.
func WithStallThreshold(d time.Duration) baseOpt {
	return func(o *options) {
		o.stallThreshold = d
	}
}

// WithStallHandler sets a custom callback for stall detection.
// If not set, stalls are logged via slog.
func WithStallHandler(h StallHandler) baseOpt {
	return func(o *options) {
		o.stallHandler = h
	}
}

// WithTimeout sets a per-invocation timeout. When combined with WithRetry,
// each retry attempt gets its own fresh deadline.
func WithTimeout(timeout time.Duration) baseOpt {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithRetry configures automatic retry with the given maximum attempts.
// maxAttempts is the total number of attempts (1 = no retry, 3 = initial + 2 retries).
func WithRetry(maxAttempts int, opts ...RetryOption) baseOpt {
	return func(o *options) {
		o.retryAttempts = maxAttempts
		o.retryOpts = opts
	}
}

// WithCircuitBreaker sets a circuit breaker for the operation. The circuit
// breaker is stateful — create one via NewCircuitBreaker and share it across
// all calls to the same dependency.
func WithCircuitBreaker(cb *CircuitBreaker) baseOpt {
	return func(o *options) {
		o.circuitBreaker = cb
	}
}

// WithFallback sets a fallback function that is called when the operation fails.
// The fallback receives the original error and may return nil to suppress it or
// a different error.
func WithFallback(fn func(ctx context.Context, err error) error, opts ...FallbackOption) baseOpt {
	return func(o *options) {
		o.fallbackFn = fn
		o.fallbackOpts = opts
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
