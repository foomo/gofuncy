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
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type (
	Options struct {
		l    *slog.Logger
		ctx  context.Context //nolint:containedctx // required
		name string
		// telemetry
		meter            metric.Meter
		tracer           trace.Tracer
		counter          metric.Int64UpDownCounter
		counterName      string
		histogram        metric.Int64Histogram
		histogramName    string
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

func WithCounterName(name string) Option {
	return func(o *Options) {
		o.counterName = name
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
		if value, err := o.meter.Int64UpDownCounter(o.counterName); err != nil {
			o.l.Error("failed to initialize gauge", "error", err)
		} else {
			o.counter = value
		}
		if value, err := o.meter.Int64Histogram(o.histogramName); err != nil {
			o.l.Error("failed to initialize histogram", "error", err)
		} else {
			o.histogram = value
		}
	}

	err := make(chan error)
	go func(o *Options, err chan<- error) {
		defer close(err)
		ctx := o.ctx
		var span trace.Span
		if o.tracer != nil {
			ctx, span = o.tracer.Start(o.ctx, o.name)
			defer span.End()
		}
		// create telemetry if enabled
		if o.counter != nil {
			o.counter.Add(ctx, 1, metric.WithAttributes(semconv.ProcessRuntimeName(o.name)))
			defer o.counter.Add(ctx, -1, metric.WithAttributes(semconv.ProcessRuntimeName(o.name)))
		}
		if o.histogram != nil {
			start := time.Now()
			defer func() {
				o.histogram.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(semconv.ProcessRuntimeName(o.name)))
			}()
		}
		ctx = injectParentRoutineIntoContext(ctx, RoutineFromContext(ctx))
		ctx = injectRoutineIntoContext(ctx, o.name)
		err <- fn(ctx)
	}(o, err)
	return err
}
