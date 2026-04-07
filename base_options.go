package gofuncy

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

type baseOptions struct {
	l            *slog.Logger
	name         string
	timeout      time.Duration
	callerSkip   int
	errorHandler ErrorHandler
	// telemetry
	tracing             bool
	startedCounter      bool
	finishedCounter     bool
	errorCounter        bool
	activeUpDownCounter bool
	durationHistogram   bool
	// middleware
	middlewares []Middleware
	// telemetry providers
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
}

func (o *baseOptions) meter() metric.Meter {
	mp := o.meterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	return mp.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL))
}

func (o *baseOptions) tracer() trace.Tracer {
	tp := o.tracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(ScopeName)
}
