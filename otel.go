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

var goroutinesCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.goroutines",
		metric.WithDescription("Gofuncy running go routine count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var goroutinesDurationHistogram = sync.OnceValue(func() metric.Int64Histogram {
	h, err := meter.Int64Histogram("gofuncy.goroutines.duration",
		metric.WithDescription("Gofuncy go routine duration histogram"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 50, 100, 500, 1000, 5000, 10000, 30000, 60000, 300000, 600000))
	if err != nil {
		otel.Handle(err)
	}

	return h
})

var chansCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.chans",
		metric.WithDescription("Gofuncy open chan count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesCounter = sync.OnceValue(func() metric.Int64UpDownCounter {
	c, err := meter.Int64UpDownCounter("gofuncy.messages",
		metric.WithDescription("Gofuncy pending message count"))
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesDurationHistogram = sync.OnceValue(func() metric.Int64Histogram {
	h, err := meter.Int64Histogram("gofuncy.messages.duration",
		metric.WithDescription("Gofuncy chan message send duration"),
		metric.WithUnit("ms"))
	if err != nil {
		otel.Handle(err)
	}

	return h
})
