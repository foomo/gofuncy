package gofuncy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/Ju0x/humanhash"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

type (
	Options struct {
		l     *slog.Logger
		ctx   context.Context //nolint:containedctx // required
		level slog.Level
		name  string
		// telemetry
		meter            metric.Meter
		tracer           trace.Tracer
		counter          metric.Int64UpDownCounter
		counterName      string
		histogram        metric.Int64Histogram
		histogramName    string
		histogramEnabled bool
		telemetryEnabled bool
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

func WithLogger(l *slog.Logger) Option {
	return func(o *Options) {
		o.l = l
	}
}

func WithLogLevel(level slog.Level) Option {
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

func WithCounterName(name string) Option {
	return func(o *Options) {
		o.counterName = name
	}
}

func WithHistogramEnabled(v bool) Option {
	return func(o *Options) {
		o.histogramEnabled = v
	}
}

func WithHistogramName(name string) Option {
	return func(o *Options) {
		o.histogramName = name
	}
}

func Go(fn Func, opts ...Option) <-chan error {
	o := &Options{
		l:                slog.Default(),
		level:            slog.LevelDebug,
		counterName:      "gofuncy.routine.count",
		histogramName:    "gofuncy.routine.duration",
		telemetryEnabled: os.Getenv("OTEL_ENABLED") == "true",
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
		if value, err := o.meter.Int64UpDownCounter(
			o.counterName,
			metric.WithDescription("Gofuncy routine counter"),
		); err != nil {
			o.l.Error("failed to initialize counter", "error", err)
		} else {
			o.counter = value
		}
	}
	if o.meter != nil && o.histogramEnabled {
		if value, err := o.meter.Int64Histogram(
			o.histogramName,
			metric.WithDescription("Gofuncy routine duration histogram"),
		); err != nil {
			o.l.Error("failed to initialize histogram", "error", err)
		} else {
			o.histogram = value
		}
	}

	err := make(chan error)
	go func(o *Options, errChan chan<- error) {
		var err error
		defer close(errChan)
		ctx := o.ctx
		start := time.Now()
		l := o.l.With("name", o.name)
		if value := RoutineFromContext(ctx); value != "" {
			l = l.With("parent", value)
		}
		var span trace.Span
		if o.tracer != nil {
			ctx, span = o.tracer.Start(o.ctx, o.name)
			l = l.With("trace_id", span.SpanContext().TraceID().String())
			defer span.End()
		}
		l.Log(ctx, o.level, "starting gofuncy routine")
		defer func() {
			if err != nil {
				l = l.With("error", err.Error())
			}
			l.Log(ctx, o.level, "exiting gofuncy routine", "duration", time.Since(start).Round(time.Millisecond).String())
		}()
		// create telemetry if enabled
		if o.counter != nil {
			attrs := metric.WithAttributes(semconv.ProcessRuntimeName(o.name))
			o.counter.Add(ctx, 1, attrs)
			defer o.counter.Add(ctx, -1, attrs)
		}
		if o.histogram != nil {
			start := time.Now()
			defer func() {
				o.histogram.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(
					semconv.ProcessRuntimeName(o.name),
					attribute.Bool("error", err != nil),
				))
			}()
		}
		ctx = injectParentRoutineIntoContext(ctx, RoutineFromContext(ctx))
		ctx = injectRoutineIntoContext(ctx, o.name)
		err = fn(ctx)
		errChan <- err
	}(o, err)
	return err
}
