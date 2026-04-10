package gofuncy

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	runtimex "github.com/foomo/go/runtime"
	"github.com/foomo/gofuncy/semconv"
	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

func withContextInjection(fn Func, name string) Func {
	return func(ctx context.Context) error {
		routineName := NameFromContext(ctx)
		hasParent := routineName != NameNoName
		hasName := name != NameNoName

		switch {
		case hasParent && hasName:
			ctx = injectRoutineIntoContext(ctx, name, routineName)
		case hasParent:
			ctx = injectParentIntoContext(ctx, routineName)
		case hasName:
			ctx = injectNameIntoContext(ctx, name)
		}

		return fn(ctx)
	}
}

func withTimeout(fn Func, timeout time.Duration) Func {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return fn(ctx)
	}
}

func withRecover(fn Func) Func {
	return func(ctx context.Context) (err error) { //nolint:nonamedreturns
		defer recoverError(&err)
		return fn(ctx)
	}
}

func withStartedCounter(fn Func, m metric.Meter, name string) Func {
	started, err := gofuncyconv.NewGoroutinesStarted(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context) error {
		started.Add(ctx, 1, name)
		return fn(ctx)
	}
}

func withErrorCounter(fn Func, m metric.Meter, name string) Func {
	errors, err := gofuncyconv.NewGoroutinesErrors(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context) error {
		err := fn(ctx)
		if err != nil {
			errors.Add(ctx, 1, name)
		}

		return err
	}
}

func withActiveUpDownCounter(fn Func, m metric.Meter, name string) Func {
	active, err := gofuncyconv.NewGoroutinesActive(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context) error {
		active.Add(ctx, 1, name)

		defer func() { active.Add(context.WithoutCancel(ctx), -1, name) }()

		return fn(ctx)
	}
}

func withDurationHistogram(fn Func, m metric.Meter, name string) Func {
	duration, err := gofuncyconv.NewGoroutinesDuration(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context) error {
		start := time.Now()
		err := fn(ctx)
		dur := time.Since(start).Truncate(time.Millisecond).Seconds()
		duration.Record(context.WithoutCancel(ctx), dur, name, err != nil)

		return err
	}
}

func withTracing(fn Func, o *options, spanPrefix string, callerSkip int) Func {
	var traceAttrsBuf [4]attribute.KeyValue

	traceAttrs := traceAttrsBuf[:0]

	if _, full, file, line, ok := runtimex.Caller(callerSkip); ok {
		traceAttrs = append(traceAttrs,
			otelsemconv.CodeFilePath(file),
			otelsemconv.CodeLineNumber(line),
			otelsemconv.CodeFunctionName(full),
		)
	}

	traceAttrs = traceAttrs[:len(traceAttrs):len(traceAttrs)]
	spanName := spanPrefix + " " + o.name

	return func(ctx context.Context) error {
		attrs := traceAttrs

		routineName := NameFromContext(ctx)
		if routineName != NameNoName {
			attrs = append(attrs, semconv.RoutineParent(routineName))
		}

		startOpts := []trace.SpanStartOption{
			trace.WithAttributes(attrs...),
		}

		if o.detachedTrace {
			if parentSpan := trace.SpanFromContext(ctx); parentSpan.SpanContext().IsValid() {
				startOpts = append(startOpts,
					trace.WithNewRoot(),
					trace.WithLinks(trace.Link{SpanContext: parentSpan.SpanContext()}),
				)
			}
		}

		ctx, span := o.tracer().Start(ctx, spanName, startOpts...)

		err := fn(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()

		return err
	}
}

func withStallDetector(fn Func, threshold time.Duration, handler StallHandler, m metric.Meter, l *slog.Logger, name string) Func {
	stalled, err := gofuncyconv.NewGoroutinesStalled(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(ctx context.Context) error {
		timer := time.AfterFunc(threshold, func() {
			stalled.Add(ctx, 1, name)

			if handler != nil {
				handler(ctx, name, threshold)
				return
			}

			if l == nil {
				l = slog.Default()
			}

			l.WarnContext(ctx, "gofuncy: stall detected", "name", name, "threshold", threshold)
		})
		defer timer.Stop()

		return fn(ctx)
	}
}

func buildChain(fn Func, o *options, spanPrefix string, callerSkip int) Func {
	run := fn
	run = withRecover(run)

	// resilience chain (innermost → outermost)
	if o.timeout > 0 {
		run = withTimeout(run, o.timeout)
	}

	if o.retryAttempts > 1 {
		opts := make([]RetryOption, len(o.retryOpts)+1)
		copy(opts, o.retryOpts)
		opts[len(o.retryOpts)] = retryWithMeter(o.meter(), o.name)

		run = Retry(o.retryAttempts, opts...)(run)
	}

	if o.circuitBreaker != nil {
		run = o.circuitBreaker.middleware(o.meter(), o.name)(run)
	}

	if o.fallbackFn != nil {
		run = Fallback(o.fallbackFn, o.fallbackOpts...)(run)
	}

	// user middlewares
	for _, m := range o.middlewares {
		run = m(run)
	}

	if o.startedCounter || o.errorCounter || o.activeUpDownCounter || o.durationHistogram {
		m := o.meter()

		if o.startedCounter {
			run = withStartedCounter(run, m, o.name)
		}

		if o.errorCounter {
			run = withErrorCounter(run, m, o.name)
		}

		if o.activeUpDownCounter {
			run = withActiveUpDownCounter(run, m, o.name)
		}

		if o.durationHistogram {
			run = withDurationHistogram(run, m, o.name)
		}
	}

	if o.tracing {
		run = withTracing(run, o, spanPrefix, callerSkip)
	}

	if o.stallThreshold > 0 {
		run = withStallDetector(run, o.stallThreshold, o.stallHandler, o.meter(), o.l, o.name)
	}

	return run
}

func handleError(ctx context.Context, err error, handler ErrorHandler, l *slog.Logger, name string) {
	if handler != nil {
		handler(ctx, err)
		return
	}

	if l == nil {
		l = slog.Default()
	}

	l.ErrorContext(ctx, "gofuncy.go error", "name", name, "err", err)
}
