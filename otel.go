package gofuncy

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter  = otel.Meter("github.com/foomo/gofuncy")
	tracer = otel.Tracer("github.com/foomo/gofuncy")
)

var goroutinesCounter = sync.OnceValue(func() metric.Int64Counter {
	c, err := meter.Int64Counter("gofuncy.goroutines.total",
		metric.WithDescription("Gofuncy running go routine count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var goroutinesUpDownCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.goroutines.current",
		metric.WithDescription("Gofuncy running go routine up/down count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var goroutinesDurationHistogram = sync.OnceValue(func() metric.Float64Histogram {
	h, err := meter.Float64Histogram("gofuncy.goroutines.duration.seconds",
		metric.WithDescription("Gofuncy go routine duration histogram"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 300.0, 600.0))
	if err != nil {
		otel.Handle(err)
	}

	return h
})

var chansUpDownCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.chans.current",
		metric.WithDescription("Gofuncy open chan up/down count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.messages.current",
		metric.WithDescription("Gofuncy pending message count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesDurationHistogram = sync.OnceValue(func() metric.Float64Histogram {
	h, err := meter.Float64Histogram("gofuncy.messages.duration.seconds",
		metric.WithDescription("Gofuncy chan message send duration"),
		metric.WithUnit("s"))
	if err != nil {
		otel.Handle(err)
	}

	return h
})
