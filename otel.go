package gofuncy

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

var (
	meter  = otel.Meter("github.com/foomo/gofuncy", metric.WithSchemaURL(otelsemconv.SchemaURL))
	tracer = otel.Tracer("github.com/foomo/gofuncy")
)

// instrumentation encapsulates all OTel metrics for goroutine operations.
type instrumentation struct {
	goroutinesTotal    gofuncyconv.GoroutinesTotal
	goroutinesCurrent  gofuncyconv.GoroutinesCurrent
	goroutinesDuration gofuncyconv.GoroutinesDuration
	groupsDuration     gofuncyconv.GroupsDuration
}

var initInstrumentation = sync.OnceValues(func() (*instrumentation, error) {
	inst := &instrumentation{}

	var err, e error

	inst.goroutinesTotal, e = gofuncyconv.NewGoroutinesTotal(meter)
	err = errors.Join(err, e)

	inst.goroutinesCurrent, e = gofuncyconv.NewGoroutinesCurrent(meter)
	err = errors.Join(err, e)

	inst.goroutinesDuration, e = gofuncyconv.NewGoroutinesDuration(meter)
	err = errors.Join(err, e)

	inst.groupsDuration, e = gofuncyconv.NewGroupsDuration(meter)
	err = errors.Join(err, e)

	return inst, err
})

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

// chan metrics — kept as sync.OnceValue for backward compatibility with chan.go
var chansCurrentMetric = sync.OnceValue(func() gofuncyconv.ChansCurrent {
	c, err := gofuncyconv.NewChansCurrent(meter)
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesCurrentMetric = sync.OnceValue(func() gofuncyconv.MessagesCurrent {
	c, err := gofuncyconv.NewMessagesCurrent(meter)
	if err != nil {
		otel.Handle(err)
	}

	return c
})

var messagesDurationMetric = sync.OnceValue(func() gofuncyconv.MessagesDuration {
	h, err := gofuncyconv.NewMessagesDuration(meter)
	if err != nil {
		otel.Handle(err)
	}

	return h
})
