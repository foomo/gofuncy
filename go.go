package gofuncy

import (
	"context"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Options struct {
		l     *zap.Logger
		level zapcore.Level
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
	Option func(*Options)
)

func WithName(name string) Option {
	return func(o *Options) {
		o.name = name
	}
}

func WithLogger(l *zap.Logger) Option {
	return func(o *Options) {
		o.l = l
	}
}

func WithLogLevel(level zapcore.Level) Option {
	return func(o *Options) {
		o.level = level
	}
}

func WithTelemetryEnabled(enabled bool) Option {
	return func(o *Options) {
		o.telemetryEnabled = enabled
	}
}

func WithMeter(meter metric.Meter) Option {
	return func(o *Options) {
		o.meter = meter
	}
}

func WithTracer(tracer trace.Tracer) Option {
	return func(o *Options) {
		o.tracer = tracer
	}
}

func WithCountMetricName(name string) Option {
	return func(o *Options) {
		o.countMetricName = name
	}
}

func WithDurationMetricName(name string) Option {
	return func(o *Options) {
		o.durationMetricName = name
	}
}

func WithDurationMetricEnabled(v bool) Option {
	return func(o *Options) {
		o.durationMetricEnabled = v
	}
}

func Go(ctx context.Context, fn Func, opts ...Option) <-chan error {
	o := &Options{
		l:                  zap.NewNop(),
		name:               NameNoName,
		level:              zapcore.DebugLevel,
		countMetricName:    "gofuncy.goroutines",
		durationMetricName: "gofuncy.goroutines.duration",
		telemetryEnabled:   os.Getenv("OTEL_ENABLED") == "true",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	// create logger
	l := o.l.Named("gofuncy.go").With(zap.String("gofuncy_name", o.name))

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
				semconv.CodeFilepath(file),
				semconv.CodeLineNumber(line),
				semconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
	}

	if o.meter != nil {
		if value, err := o.meter.Int64UpDownCounter(
			o.countMetricName,
			metric.WithDescription("Gofuncy running go routine count"),
		); err != nil {
			l.Warn("failed to initialize counter", zap.Error(err))
		} else {
			o.runningMetric = value
		}
	}

	if o.meter != nil && o.durationMetricEnabled {
		if value, err := o.meter.Int64Histogram(
			o.durationMetricName,
			metric.WithDescription("Gofuncy go routine duration histogram"),
		); err != nil {
			l.Warn("failed to initialize histogram", zap.Error(err))
		} else {
			o.durationMetric = value
		}
	}

	delay := time.Now()
	errChan := make(chan error, 1)
	go func(ctx context.Context, o *Options, errChan chan<- error) {
		defer close(errChan)

		if ctx.Err() != nil {
			errChan <- ctx.Err()
			return
		}

		var err error
		start := time.Now()
		routineName := NameFromContext(ctx)

		if routineName != NameNoName {
			l = l.With(zap.String("gofuncy_parent", routineName))
			traceAttrs = append(traceAttrs, attribute.String("gofuncy.parent", routineName))
		}

		var span trace.Span
		if o.tracer != nil {
			ctx, span = o.tracer.Start(ctx,
				"GOFUNCY go."+o.name,
				trace.WithAttributes(traceAttrs...),
			)
			if span.IsRecording() {
				l = l.With(
					zap.String("trace_id", span.SpanContext().TraceID().String()),
					zap.String("span_id", span.SpanContext().SpanID().String()),
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

		l.Log(o.level, "go",
			zap.Duration("delay", time.Since(delay).Round(time.Millisecond)),
		)
		defer func() {
			l.Log(o.level, "stop",
				zap.Duration("duration", time.Since(start).Round(time.Millisecond)),
				zap.Error(err),
			)
		}()
		// create telemetry if enabled
		metricAttrs := metric.WithAttributes(semconv.ProcessRuntimeName(o.name))
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
		ctx = injectParentIntoContext(ctx, NameFromContext(ctx))
		ctx = injectNameIntoContext(ctx, o.name)
		err = fn(ctx)
		errChan <- err
	}(ctx, o, errChan)

	return errChan
}
