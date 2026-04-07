package gofuncy

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

// Async spawns a single goroutine and returns its error channel.
// The caller has explicit lifecycle control by reading from the returned channel.
func Async(ctx context.Context, fn Func, opts ...Option[AsyncOptions]) <-chan error {
	o := newAsyncOptions(opts)

	// create logger (only if provided, avoid allocation)
	l := o.l
	if l != nil {
		l = l.WithGroup("gofuncy.async").With("gofuncy_name", o.name)
	}

	// create telemetry if enabled — use stack-allocated array to avoid heap alloc
	var traceAttrsBuf [4]attribute.KeyValue

	traceAttrs := traceAttrsBuf[:0]

	if o.tracing {
		if pc, file, line, ok := runtime.Caller(1); ok {
			traceAttrs = append(traceAttrs,
				otelsemconv.CodeFilePath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
	}

	// only capture delay when logger is set
	var delay time.Time
	if l != nil {
		delay = time.Now()
	}

	errChan := make(chan error, 1)
	go func(ctx context.Context, o *AsyncOptions, errChan chan<- error) {
		defer close(errChan)

		var err error

		defer func() { errChan <- err }()

		defer recoverError(&err)

		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}

		start := time.Now()
		routineName := NameFromContext(ctx)

		if routineName != NameNoName {
			if l != nil {
				l = l.With("gofuncy_parent", routineName)
			}

			traceAttrs = append(traceAttrs, semconv.RoutineParent(routineName))
		}

		var span trace.Span

		if o.tracing {
			ctx, span = resolveTracer(o.tracerProvider).Start(ctx,
				"gofuncy.async "+o.name,
				trace.WithAttributes(traceAttrs...),
			)

			if span.IsRecording() && l != nil {
				l = l.With(
					"trace_id", span.SpanContext().TraceID().String(),
					"span_id", span.SpanContext().SpanID().String(),
				)
			}

			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}

				span.End()
			}()
		}

		if l != nil {
			l.DebugContext(ctx, "go",
				"delay", time.Since(delay).Round(time.Millisecond),
			)
		}

		defer func() {
			if l != nil {
				if err != nil {
					l.WarnContext(ctx, "stop",
						"duration", time.Since(start).Round(time.Millisecond),
						"err", err,
					)
				} else {
					l.DebugContext(ctx, "stop",
						"duration", time.Since(start).Round(time.Millisecond),
					)
				}
			}
		}()

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
	}(ctx, o, errChan)

	return errChan
}

// AsyncBackground is like Async but detaches from the parent context's cancellation.
// The goroutine will continue running even if the parent context is canceled.
func AsyncBackground(ctx context.Context, fn Func, opts ...Option[AsyncOptions]) <-chan error {
	return Async(context.WithoutCancel(ctx), fn, opts...)
}
