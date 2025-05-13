package gofuncy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Ju0x/humanhash"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Options struct {
		l     *zap.Logger
		ctx   context.Context //nolint:containedctx // required
		level zapcore.Level
		name  string
		// telemetry
		meter                    metric.Meter
		tracer                   trace.Tracer
		totalCounter             metric.Int64Counter
		totalCounterName         string
		runningCounter           metric.Int64UpDownCounter
		runningCounterName       string
		durationHistogram        metric.Int64Histogram
		durationHistogramName    string
		durationHistogramEnabled bool
		telemetryEnabled         bool
	}
	Option func(*Options)
)

func WithName(name string) Option {
	return func(o *Options) {
		o.name = name
	}
}

func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
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

func WithMeter(v metric.Meter) Option {
	return func(o *Options) {
		o.meter = v
	}
}

func WithTracer(v trace.Tracer) Option {
	return func(o *Options) {
		o.tracer = v
	}
}

func WithTotalCounterName(name string) Option {
	return func(o *Options) {
		o.totalCounterName = name
	}
}

func WithRunningCounterName(name string) Option {
	return func(o *Options) {
		o.runningCounterName = name
	}
}

func WithDurationHistogramEnabled(v bool) Option {
	return func(o *Options) {
		o.durationHistogramEnabled = v
	}
}

func WithHistogramName(name string) Option {
	return func(o *Options) {
		o.durationHistogramName = name
	}
}

func Go(fn Func, opts ...Option) <-chan error {
	o := &Options{
		l:                     zap.NewNop(),
		level:                 zapcore.DebugLevel,
		totalCounterName:      "gofuncy.routine.total.count",
		runningCounterName:    "gofuncy.routine.running.count",
		durationHistogramName: "gofuncy.routine.duration",
		telemetryEnabled:      os.Getenv("OTEL_ENABLED") == "true",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	if o.ctx == nil {
		o.ctx = context.Background()
	}
	if o.name == "" {
		if _, file, line, ok := runtime.Caller(0); ok {
			h := sha256.New()
			_, _ = fmt.Fprintf(h, "%s:%d", file, line)
			o.name, _ = humanhash.Humanize(h.Sum(nil), 2, "-")
		}
	}
	// create telemetry if enabled
	if o.telemetryEnabled {
		if o.meter == nil {
			o.meter = otel.Meter("gofuncy")
		}
		if o.tracer == nil {
			o.tracer = otel.Tracer("gofuncy")
		}
	}
	if o.meter != nil {
		if value, err := o.meter.Int64Counter(
			o.totalCounterName,
			metric.WithDescription("Gofuncy total go routine count"),
		); err != nil {
			o.l.Error("failed to initialize counter", zap.Error(err))
		} else {
			o.totalCounter = value
		}
		if value, err := o.meter.Int64UpDownCounter(
			o.runningCounterName,
			metric.WithDescription("Gofuncy running go routine count"),
		); err != nil {
			o.l.Error("failed to initialize counter", zap.Error(err))
		} else {
			o.runningCounter = value
		}
	}
	if o.meter != nil && o.durationHistogramEnabled {
		if value, err := o.meter.Int64Histogram(
			o.durationHistogramName,
			metric.WithDescription("Gofuncy go routine duration histogram"),
		); err != nil {
			o.l.Error("failed to initialize histogram", zap.Error(err))
		} else {
			o.durationHistogram = value
		}
	}

	delay := time.Now()
	errChan := make(chan error, 1)
	go func(o *Options, errChan chan<- error) {
		var err error
		ctx := o.ctx
		start := time.Now()
		defer close(errChan)
		l := o.l.With(zap.String("name", o.name))
		if value := RoutineFromContext(ctx); value != NoNameRoutine {
			l = l.With(zap.String("parent", value))
		}
		var span trace.Span
		if o.tracer != nil {
			ctx, span = o.tracer.Start(o.ctx, o.name)
			if span.IsRecording() {
				l = l.With(zap.String("trace_id", span.SpanContext().TraceID().String()))
			}
			defer span.End()
		}
		l.Log(o.level, "starting gofuncy routine",
			zap.Duration("delay", time.Since(delay).Round(time.Millisecond)),
		)
		defer func() {
			l.Log(o.level, "exiting gofuncy routine",
				zap.Duration("duration", time.Since(start).Round(time.Millisecond)),
				zap.Error(err),
			)
		}()
		// create telemetry if enabled
		attrs := metric.WithAttributes(semconv.ProcessRuntimeName(o.name))
		if o.runningCounter != nil {
			o.runningCounter.Add(ctx, 1, attrs)
			defer o.runningCounter.Add(ctx, -1, attrs)
		}
		if o.totalCounter != nil {
			o.totalCounter.Add(ctx, 1, attrs)
			defer o.runningCounter.Add(ctx, -1, attrs, metric.WithAttributes(
				attribute.Bool("error", err != nil),
			))
		}
		if o.durationHistogram != nil {
			defer func() {
				o.durationHistogram.Record(ctx, time.Since(start).Milliseconds(), attrs, metric.WithAttributes(
					attribute.Bool("error", err != nil),
				))
			}()
		}
		ctx = injectParentRoutineIntoContext(ctx, RoutineFromContext(ctx))
		ctx = injectRoutineIntoContext(ctx, o.name)
		err = fn(ctx)
		errChan <- err
	}(o, errChan)

	return errChan
}
