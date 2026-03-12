package gofuncy

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	otelSemconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

var (
	defaultTelemetryEnabled         = os.Getenv("GOFUNCY_TELEMETRY_ENABLED") == "true"
	defaultMessagesAttributeEnabled = os.Getenv("GOFUNCY_MESSAGES_ATTRIBUTE_ENABLED") == "true"
	optionsPool                     = sync.Pool{New: func() any { return &options{} }}
)

type (
	options struct {
		l     *slog.Logger
		level slog.Level
		name  string
		// telemetry
		meter                 metric.Meter
		tracer                trace.Tracer
		runningMetric         metric.Int64UpDownCounter
		countMetricName       string
		durationMetric        metric.Int64Histogram
		durationMetricName    string
		durationMetricEnabled bool
		telemetryEnabled      bool
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

func WithLogLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

func WithTelemetryEnabled(enabled bool) Option {
	return func(o *options) {
		o.telemetryEnabled = enabled
	}
}

func WithMeter(meter metric.Meter) Option {
	return func(o *options) {
		o.meter = meter
	}
}

func WithTracer(tracer trace.Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

func WithCountMetricName(name string) Option {
	return func(o *options) {
		o.countMetricName = name
	}
}

func WithDurationMetricName(name string) Option {
	return func(o *options) {
		o.durationMetricName = name
	}
}

func WithDurationMetricEnabled(v bool) Option {
	return func(o *options) {
		o.durationMetricEnabled = v
	}
}

// reset clears all fields in options for reuse from pool (OPT 6)
func (o *options) reset() {
	o.l = nil
	o.level = slog.LevelDebug
	o.name = NameNoName
	o.meter = nil
	o.tracer = nil
	o.runningMetric = nil
	o.countMetricName = "gofuncy.goroutines"
	o.durationMetric = nil
	o.durationMetricName = "gofuncy.goroutines.duration"
	o.durationMetricEnabled = false
	o.telemetryEnabled = defaultTelemetryEnabled
}

func Go(ctx context.Context, fn Func, opts ...Option) <-chan error {
	// OPT 6: get options from pool, reset to defaults
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

	// create telemetry if enabled
	var traceAttrs []attribute.KeyValue

	if o.telemetryEnabled {
		if o.meter == nil {
			o.meter = otel.Meter("github.com/foomo/gofuncy")
		}

		if o.tracer == nil {
			o.tracer = otel.Tracer("github.com/foomo/gofuncy")
		}
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			traceAttrs = append(traceAttrs,
				otelSemconv.CodeFilepath(file),
				otelSemconv.CodeLineNumber(line),
				otelSemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
	}

	if o.meter != nil {
		if value, err := o.meter.Int64UpDownCounter(
			o.countMetricName,
			metric.WithDescription("Gofuncy running go routine count"),
		); err != nil {
			if l != nil {
				l.Warn("failed to initialize counter", "err", err)
			}
		} else {
			o.runningMetric = value
		}
	}

	if o.meter != nil && o.durationMetricEnabled {
		if value, err := o.meter.Int64Histogram(
			o.durationMetricName,
			metric.WithDescription("Gofuncy go routine duration histogram"),
			metric.WithUnit("ms"),
		); err != nil {
			if l != nil {
				l.Warn("failed to initialize histogram", "err", err)
			}
		} else {
			o.durationMetric = value
		}
	}

	delay := time.Now()

	errChan := make(chan error, 1)
	go func(ctx context.Context, o *options, errChan chan<- error) {
		defer close(errChan)
		defer optionsPool.Put(o) // OPT 6: return to pool when done

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

			traceAttrs = append(traceAttrs, attribute.String("gofuncy.routine.parent", routineName))
		}

		var span trace.Span
		if o.tracer != nil {
			ctx, span = o.tracer.Start(ctx,
				"gofuncy.go "+o.name,
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
			l.Log(ctx, o.level, "go",
				"delay", time.Since(delay).Round(time.Millisecond),
			)
		}

		defer func() {
			if l != nil {
				if err != nil {
					l.Log(ctx, o.level, "stop",
						"duration", time.Since(start).Round(time.Millisecond),
						"err", err,
					)
				} else {
					l.Log(ctx, o.level, "stop",
						"duration", time.Since(start).Round(time.Millisecond),
					)
				}
			}
		}()
		// create telemetry if enabled (guard to avoid alloc when metrics are nil)
		if o.runningMetric != nil || o.durationMetric != nil {
			metricAttrs := metric.WithAttributes(semconv.RoutineName.String(o.name))
			if o.runningMetric != nil {
				o.runningMetric.Add(ctx, 1, metricAttrs)
				defer o.runningMetric.Add(ctx, -1, metricAttrs)
			}

			if o.durationMetric != nil {
				defer func() {
					o.durationMetric.Record(ctx, time.Since(start).Milliseconds(), metricAttrs, metric.WithAttributes(
						attribute.Bool("error", err != nil),
					))
				}()
			}
		}

		// OPT 3: reuse routineName from earlier lookup instead of calling NameFromContext again
		if routineName != NameNoName {
			ctx = injectParentIntoContext(ctx, routineName)
		}

		if o.name != NameNoName {
			ctx = injectNameIntoContext(ctx, o.name)
		}

		err = fn(ctx)
		errChan <- err
	}(ctx, o, errChan)

	return errChan
}
