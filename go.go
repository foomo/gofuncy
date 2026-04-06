package gofuncy

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

// Go spawns a fire-and-forget goroutine with panic recovery.
// Errors are logged via slog by default; use GoOption().WithErrorHandler to override.
func Go(ctx context.Context, fn Func, opts ...Option[GoOptions]) {
	o := newGoOptions(opts)

	// capture error handler and logger before passing options to goroutine
	errorHandler := o.errorHandler
	l := o.l

	// create telemetry if enabled — use stack-allocated array to avoid heap alloc
	var traceAttrsBuf [4]attribute.KeyValue

	traceAttrs := traceAttrsBuf[:0]

	if o.tracing {
		if pc, file, line, ok := runtime.Caller(o.callerSkip + 1); ok {
			traceAttrs = append(traceAttrs,
				otelsemconv.CodeFilePath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
	}

	go func(ctx context.Context, o *GoOptions) {
		var (
			err           error
			cancel        context.CancelFunc
			cancelTimeout context.CancelFunc
		)

		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		if o.timeout > 0 {
			ctx, cancelTimeout = context.WithTimeout(ctx, o.timeout)
			defer cancelTimeout()
		}

		defer func(ctx context.Context) {
			if err == nil {
				return
			}

			if errorHandler != nil {
				errorHandler(ctx, err)

				return
			}

			if l != nil {
				l.ErrorContext(ctx, "gofuncy.go error",
					"name", o.name,
					"err", err,
				)

				return
			}

			slog.Default().WarnContext(ctx, "gofuncy.go error",
				"name", o.name,
				"err", err,
			)
		}(ctx)

		start := time.Now()
		routineName := NameFromContext(ctx)

		if routineName != NameNoName {
			traceAttrs = append(traceAttrs, semconv.RoutineParent(routineName))
		}

		var span trace.Span

		if o.tracing {
			ctx, span = resolveTracer(o.tracerProvider).Start(ctx,
				"gofuncy.go "+o.name,
				trace.WithAttributes(traceAttrs...),
			)

			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}

				span.End()
			}()
		}

		defer recoverError(&err)

		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}

		// create metrics if enabled
		if o.upDownMetric || o.counterMetric || o.durationMetric {
			inst := resolveInstrumentation(o.meterProvider)

			if o.upDownMetric {
				inst.incGoroutine(ctx, o.name)

				defer func() {
					inst.decGoroutine(ctx, o.name)
				}()
			}

			if o.counterMetric {
				inst.addGoroutine(ctx, o.name)
			}

			if o.durationMetric {
				defer func() {
					dur := time.Since(start).Truncate(time.Millisecond).Seconds()
					inst.recordGoroutineDuration(context.WithoutCancel(ctx), dur, o.name, err != nil)
				}()
			}
		}

		// combine parent + name into single context.WithValue when both are needed
		hasParent := routineName != NameNoName
		hasName := o.name != NameNoName

		switch {
		case hasParent && hasName:
			ctx = injectRoutineIntoContext(ctx, o.name, routineName)
		case hasParent:
			ctx = injectParentIntoContext(ctx, routineName)
		case hasName:
			ctx = injectNameIntoContext(ctx, o.name)
		}

		err = fn(ctx)
	}(ctx, o)
}
