package gofuncy

import (
	"log/slog"
	"time"
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
}

type concurrentOptions struct {
	baseOptions
	limit    int
	failFast bool
}
