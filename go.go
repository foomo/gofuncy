package gofuncy

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

var (
	optionsPool = sync.Pool{New: func() any { return &options{} }}
)

type (
	options struct {
		l    *slog.Logger
		name string
		// telemetry
		tracingEnabled        bool
		counterMetricEnabled  bool
		durationMetricEnabled bool
	}
	Option func(*options)
)

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
		o.tracingEnabled = true
	}
}

func WithCounterMetric() Option {
	return func(o *options) {
		o.counterMetricEnabled = true
	}
}

func WithDurationMetric() Option {
	return func(o *options) {
		o.durationMetricEnabled = true
	}
}

// reset clears all fields in options for reuse from pool (OPT 6)
func (o *options) reset() {
	o.l = nil
	o.name = NameNoName
	o.tracingEnabled = false
	o.counterMetricEnabled = false
	o.durationMetricEnabled = false
}

func Go(ctx context.Context, fn Func, opts ...Option) <-chan error {
	// get options from pool, reset to defaults
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

	// create logger (only if provided, avoid allocation)
	var l *slog.Logger
	if o.l != nil {
		l = o.l.WithGroup("gofuncy.go").With("gofuncy_name", o.name)
	}

	// create telemetry if enabled — use stack-allocated array to avoid heap alloc
	var traceAttrsBuf [4]attribute.KeyValue

	traceAttrs := traceAttrsBuf[:0]

	if o.tracingEnabled {
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			traceAttrs = append(traceAttrs,
				otelsemconv.CodeFilepath(file),
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
	go func(ctx context.Context, o *options, errChan chan<- error) {
		defer close(errChan)
		defer optionsPool.Put(o)

		if ctx.Err() != nil {
			errChan <- ctx.Err()
			return
		}

		var err error

		start := time.Now()
		routineName := NameFromContext(ctx)

		if routineName != NameNoName {
			if l != nil {
				l = l.With("gofuncy_parent", routineName)
			}

			traceAttrs = append(traceAttrs, semconv.RoutineParent(routineName))
		}

		var span trace.Span

		if o.tracingEnabled {
			var sb strings.Builder
			sb.WriteString("gofuncy.go ")
			sb.WriteString(o.name)

			ctx, span = tracer.Start(ctx,
				sb.String(),
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

		// create metrics if enabled (guard to avoid alloc when metrics are nil)
		if o.counterMetricEnabled || o.durationMetricEnabled {
			metricAttrs := metric.WithAttributes(semconv.RoutineName(o.name))

			if o.counterMetricEnabled {
				counter := goroutinesCounter()
				counter.Add(ctx, 1, metricAttrs)

				defer func() {
					counter.Add(ctx, -1, metricAttrs)
				}()
			}

			if o.durationMetricEnabled {
				// pre-compute error attribute options to avoid alloc in defer
				metricAttrsOk := metric.WithAttributes(semconv.RoutineName(o.name), attribute.Bool("error", false))
				metricAttrsErr := metric.WithAttributes(semconv.RoutineName(o.name), attribute.Bool("error", true))

				defer func() {
					if err != nil {
						goroutinesDurationHistogram().Record(ctx, time.Since(start).Truncate(time.Millisecond).Seconds(), metricAttrsErr)
					} else {
						goroutinesDurationHistogram().Record(ctx, time.Since(start).Truncate(time.Millisecond).Seconds(), metricAttrsOk)
					}
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
		errChan <- err
	}(ctx, o, errChan)

	return errChan
}
