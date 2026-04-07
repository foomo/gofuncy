package gofuncy

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

const ScopeName = "github.com/foomo/gofuncy"

func resolveChansCurrent(mp metric.MeterProvider) gofuncyconv.ChansCurrent {
	m := mp
	if m == nil {
		m = otel.GetMeterProvider()
	}

	c, err := gofuncyconv.NewChansCurrent(m.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL)))
	if err != nil {
		otel.Handle(err)
	}

	return c
}

func resolveMessagesCurrent(mp metric.MeterProvider) gofuncyconv.MessagesCurrent {
	m := mp
	if m == nil {
		m = otel.GetMeterProvider()
	}

	c, err := gofuncyconv.NewMessagesCurrent(m.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL)))
	if err != nil {
		otel.Handle(err)
	}

	return c
}

func resolveMessagesDuration(mp metric.MeterProvider) gofuncyconv.MessagesDuration {
	m := mp
	if m == nil {
		m = otel.GetMeterProvider()
	}

	h, err := gofuncyconv.NewMessagesDuration(m.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL)))
	if err != nil {
		otel.Handle(err)
	}

	return h
}
