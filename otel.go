package gofuncy

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

const ScopeName = "github.com/foomo/gofuncy"

// instrumentation encapsulates all OTel metrics for goroutine operations.
type instrumentation struct {
	goroutinesTotal    gofuncyconv.GoroutinesTotal
	goroutinesCurrent  gofuncyconv.GoroutinesCurrent
	goroutinesDuration gofuncyconv.GoroutinesDuration
	groupsDuration     gofuncyconv.GroupsDuration
}

func resolveMeter(mp metric.MeterProvider) metric.Meter {
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	return mp.Meter(ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL))
}

func resolveTracer(tp trace.TracerProvider) trace.Tracer {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(ScopeName)
}

func resolveInstrumentation(mp metric.MeterProvider) *instrumentation {
	inst := &instrumentation{}
	m := resolveMeter(mp)

	var err, e error

	inst.goroutinesTotal, e = gofuncyconv.NewGoroutinesTotal(m)
	err = errors.Join(err, e)

	inst.goroutinesCurrent, e = gofuncyconv.NewGoroutinesCurrent(m)
	err = errors.Join(err, e)

	inst.goroutinesDuration, e = gofuncyconv.NewGoroutinesDuration(m)
	err = errors.Join(err, e)

	inst.groupsDuration, e = gofuncyconv.NewGroupsDuration(m)
	err = errors.Join(err, e)

	if err != nil {
		otel.Handle(err)
	}

	return inst
}

func (i *instrumentation) addGoroutine(ctx context.Context, routineName string) {
	if i == nil {
		return
	}

	i.goroutinesTotal.Add(ctx, 1, routineName)
}

func (i *instrumentation) incGoroutine(ctx context.Context, routineName string) {
	if i == nil {
		return
	}

	i.goroutinesCurrent.Add(ctx, 1, routineName)
}

func (i *instrumentation) decGoroutine(ctx context.Context, routineName string) {
	if i == nil {
		return
	}

	i.goroutinesCurrent.Add(ctx, -1, routineName)
}

func (i *instrumentation) recordGoroutineDuration(ctx context.Context, seconds float64, routineName string, hasError bool) {
	if i == nil {
		return
	}

	i.goroutinesDuration.Record(ctx, seconds, routineName, hasError)
}

func (i *instrumentation) recordGroupDuration(ctx context.Context, seconds float64, routineName string, hasError bool, groupSize int) {
	if i == nil {
		return
	}

	i.groupsDuration.Record(ctx, seconds, routineName, hasError, groupSize)
}

func resolveChansCurrent(mp metric.MeterProvider) gofuncyconv.ChansCurrent {
	c, err := gofuncyconv.NewChansCurrent(resolveMeter(mp))
	if err != nil {
		otel.Handle(err)
	}

	return c
}

func resolveMessagesCurrent(mp metric.MeterProvider) gofuncyconv.MessagesCurrent {
	c, err := gofuncyconv.NewMessagesCurrent(resolveMeter(mp))
	if err != nil {
		otel.Handle(err)
	}

	return c
}

func resolveMessagesDuration(mp metric.MeterProvider) gofuncyconv.MessagesDuration {
	h, err := gofuncyconv.NewMessagesDuration(resolveMeter(mp))
	if err != nil {
		otel.Handle(err)
	}

	return h
}
