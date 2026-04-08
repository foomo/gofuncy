package gofuncy

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

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
	started, _ := gofuncyconv.NewGoroutinesStarted(m)

	return func(ctx context.Context) error {
		started.Add(ctx, 1, name)
		return fn(ctx)
	}
}

func withFinishedCounter(fn Func, m metric.Meter, name string) Func {
	finished, _ := gofuncyconv.NewGoroutinesFinished(m)

	return func(ctx context.Context) error {
		err := fn(ctx)
		finished.Add(ctx, 1, name)

		return err
	}
}

func withErrorCounter(fn Func, m metric.Meter, name string) Func {
	errors, _ := gofuncyconv.NewGoroutinesErrors(m)

	return func(ctx context.Context) error {
		err := fn(ctx)
		if err != nil {
			errors.Add(ctx, 1, name)
		}

		return err
	}
}

func withActiveUpDownCounter(fn Func, m metric.Meter, name string) Func {
	active, _ := gofuncyconv.NewGoroutinesActive(m)

	return func(ctx context.Context) error {
		active.Add(ctx, 1, name)

		defer func() { active.Add(ctx, -1, name) }()

		return fn(ctx)
	}
}

func withDurationHistogram(fn Func, m metric.Meter, name string) Func {
	duration, _ := gofuncyconv.NewGoroutinesDuration(m)

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

	if pc, file, line, ok := runtime.Caller(callerSkip); ok {
		traceAttrs = append(traceAttrs,
			otelsemconv.CodeFilePath(file),
			otelsemconv.CodeLineNumber(line),
			otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
		)
	}

	return func(ctx context.Context) error {
		routineName := NameFromContext(ctx)
		if routineName != NameNoName {
			traceAttrs = append(traceAttrs, semconv.RoutineParent(routineName))
		}

		ctx, span := o.tracer().Start(ctx,
			spanPrefix+" "+o.name,
			trace.WithAttributes(traceAttrs...),
		)

		err := fn(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()

		return err
	}
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
