package gofuncy

import (
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type baseOptions struct {
	l            *slog.Logger
	name         string
	timeout      time.Duration
	callerSkip   int
	errorHandler ErrorHandler
	// telemetry
	tracing        bool
	upDownMetric   bool
	counterMetric  bool
	durationMetric bool
	// telemetry providers
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
}

type concurrentOptions struct {
	baseOptions
	limit    int
	failFast bool
}
