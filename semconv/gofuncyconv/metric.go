package gofuncyconv

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/foomo/gofuncy/semconv"
)

// ------------------------------------------------------------------------------------------------
// ~ Constants
// ------------------------------------------------------------------------------------------------

const (
	goroutinesTotalName = "gofuncy.goroutines.total"
	goroutinesTotalDesc = "Gofuncy running go routine count"

	goroutinesCurrentName = "gofuncy.goroutines.current"
	goroutinesCurrentDesc = "Gofuncy running go routine up/down count"

	goroutinesDurationName = "gofuncy.goroutines.duration.seconds"
	goroutinesDurationDesc = "Gofuncy go routine duration histogram"

	groupsDurationName = "gofuncy.groups.duration.seconds"
	groupsDurationDesc = "Gofuncy group/map duration histogram"

	chansCurrentName = "gofuncy.chans.current"
	chansCurrentDesc = "Gofuncy open chan up/down count"

	messagesCurrentName = "gofuncy.messages.current"
	messagesCurrentDesc = "Gofuncy pending message count"

	messagesDurationName = "gofuncy.messages.duration.seconds"
	messagesDurationDesc = "Gofuncy chan message send duration"

	unitGoroutine = "{goroutine}"
	unitSeconds   = "s"
	unitChan      = "{chan}"
	unitMessage   = "{message}"
)

var (
	addOptionPool = sync.Pool{New: func() any {
		s := make([]metric.AddOption, 0, 4)
		return &s
	}}
	recordOptionPool = sync.Pool{New: func() any {
		s := make([]metric.RecordOption, 0, 4)
		return &s
	}}
)

func getAddOptions(attrs []attribute.KeyValue, extra ...attribute.KeyValue) []metric.AddOption {
	p, ok := addOptionPool.Get().(*[]metric.AddOption)
	if !ok {
		p = &[]metric.AddOption{}
	}

	opts := (*p)[:0]
	opts = append(opts, metric.WithAttributes(append(attrs, extra...)...))

	return opts
}

func putAddOptions(opts []metric.AddOption) {
	addOptionPool.Put(&opts)
}

func getRecordOptions(attrs []attribute.KeyValue, extra ...attribute.KeyValue) []metric.RecordOption {
	p, ok := recordOptionPool.Get().(*[]metric.RecordOption)
	if !ok {
		p = &[]metric.RecordOption{}
	}

	opts := (*p)[:0]
	opts = append(opts, metric.WithAttributes(append(attrs, extra...)...))

	return opts
}

func putRecordOptions(opts []metric.RecordOption) {
	recordOptionPool.Put(&opts)
}

// default histogram bucket boundaries for goroutine/group durations
var durationBuckets = metric.WithExplicitBucketBoundaries(
	0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 300.0, 600.0,
)

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesTotal
// ------------------------------------------------------------------------------------------------

// GoroutinesTotal counts the total number of goroutines spawned.
type GoroutinesTotal struct {
	inst metric.Int64Counter
}

func NewGoroutinesTotal(m metric.Meter) (GoroutinesTotal, error) {
	if m == nil {
		return GoroutinesTotal{}, nil
	}

	c, err := m.Int64Counter(goroutinesTotalName,
		metric.WithDescription(goroutinesTotalDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesTotal{inst: c}, err
}

func (GoroutinesTotal) Name() string                { return goroutinesTotalName }
func (GoroutinesTotal) Unit() string                { return unitGoroutine }
func (GoroutinesTotal) Description() string         { return goroutinesTotalDesc }
func (g GoroutinesTotal) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesTotal) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{semconv.RoutineName(routineName)}
	opts := getAddOptions(base, attrs...)

	g.inst.Add(ctx, incr, opts...)

	putAddOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesCurrent
// ------------------------------------------------------------------------------------------------

// GoroutinesCurrent tracks the current number of active goroutines.
type GoroutinesCurrent struct {
	inst metric.Int64UpDownCounter
}

func NewGoroutinesCurrent(m metric.Meter) (GoroutinesCurrent, error) {
	if m == nil {
		return GoroutinesCurrent{}, nil
	}

	c, err := m.Int64UpDownCounter(goroutinesCurrentName,
		metric.WithDescription(goroutinesCurrentDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesCurrent{inst: c}, err
}

func (GoroutinesCurrent) Name() string                      { return goroutinesCurrentName }
func (GoroutinesCurrent) Unit() string                      { return unitGoroutine }
func (GoroutinesCurrent) Description() string               { return goroutinesCurrentDesc }
func (g GoroutinesCurrent) Inst() metric.Int64UpDownCounter { return g.inst }

func (g GoroutinesCurrent) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{semconv.RoutineName(routineName)}
	opts := getAddOptions(base, attrs...)

	g.inst.Add(ctx, incr, opts...)

	putAddOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesDuration
// ------------------------------------------------------------------------------------------------

// GoroutinesDuration records the duration of individual goroutines.
type GoroutinesDuration struct {
	inst metric.Float64Histogram
}

func NewGoroutinesDuration(m metric.Meter) (GoroutinesDuration, error) {
	if m == nil {
		return GoroutinesDuration{}, nil
	}

	h, err := m.Float64Histogram(goroutinesDurationName,
		metric.WithDescription(goroutinesDurationDesc),
		metric.WithUnit(unitSeconds),
		durationBuckets,
	)

	return GoroutinesDuration{inst: h}, err
}

func (GoroutinesDuration) Name() string                    { return goroutinesDurationName }
func (GoroutinesDuration) Unit() string                    { return unitSeconds }
func (GoroutinesDuration) Description() string             { return goroutinesDurationDesc }
func (g GoroutinesDuration) Inst() metric.Float64Histogram { return g.inst }

func (g GoroutinesDuration) Record(ctx context.Context, value float64, routineName string, hasError bool, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{
		semconv.RoutineName(routineName),
		attribute.Bool("error", hasError),
	}
	opts := getRecordOptions(base, attrs...)

	g.inst.Record(ctx, value, opts...)

	putRecordOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ GroupsDuration
// ------------------------------------------------------------------------------------------------

// GroupsDuration records the duration of group/map operations.
type GroupsDuration struct {
	inst metric.Float64Histogram
}

func NewGroupsDuration(m metric.Meter) (GroupsDuration, error) {
	if m == nil {
		return GroupsDuration{}, nil
	}

	h, err := m.Float64Histogram(groupsDurationName,
		metric.WithDescription(groupsDurationDesc),
		metric.WithUnit(unitSeconds),
		durationBuckets,
	)

	return GroupsDuration{inst: h}, err
}

func (GroupsDuration) Name() string                    { return groupsDurationName }
func (GroupsDuration) Unit() string                    { return unitSeconds }
func (GroupsDuration) Description() string             { return groupsDurationDesc }
func (g GroupsDuration) Inst() metric.Float64Histogram { return g.inst }

func (g GroupsDuration) Record(ctx context.Context, value float64, routineName string, hasError bool, groupSize int, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{
		semconv.RoutineName(routineName),
		attribute.Bool("error", hasError),
		semconv.GroupSize(groupSize),
	}
	opts := getRecordOptions(base, attrs...)

	g.inst.Record(ctx, value, opts...)

	putRecordOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ ChansCurrent
// ------------------------------------------------------------------------------------------------

// ChansCurrent tracks the current number of open channels.
type ChansCurrent struct {
	inst metric.Int64UpDownCounter
}

func NewChansCurrent(m metric.Meter) (ChansCurrent, error) {
	if m == nil {
		return ChansCurrent{}, nil
	}

	c, err := m.Int64UpDownCounter(chansCurrentName,
		metric.WithDescription(chansCurrentDesc),
		metric.WithUnit(unitChan),
	)

	return ChansCurrent{inst: c}, err
}

func (ChansCurrent) Name() string                      { return chansCurrentName }
func (ChansCurrent) Unit() string                      { return unitChan }
func (ChansCurrent) Description() string               { return chansCurrentDesc }
func (g ChansCurrent) Inst() metric.Int64UpDownCounter { return g.inst }

func (g ChansCurrent) Add(ctx context.Context, incr int64, chanName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{semconv.ChanName(chanName)}
	opts := getAddOptions(base, attrs...)

	g.inst.Add(ctx, incr, opts...)

	putAddOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ MessagesCurrent
// ------------------------------------------------------------------------------------------------

// MessagesCurrent tracks the current number of pending messages.
type MessagesCurrent struct {
	inst metric.Int64UpDownCounter
}

func NewMessagesCurrent(m metric.Meter) (MessagesCurrent, error) {
	if m == nil {
		return MessagesCurrent{}, nil
	}

	c, err := m.Int64UpDownCounter(messagesCurrentName,
		metric.WithDescription(messagesCurrentDesc),
		metric.WithUnit(unitMessage),
	)

	return MessagesCurrent{inst: c}, err
}

func (MessagesCurrent) Name() string                      { return messagesCurrentName }
func (MessagesCurrent) Unit() string                      { return unitMessage }
func (MessagesCurrent) Description() string               { return messagesCurrentDesc }
func (g MessagesCurrent) Inst() metric.Int64UpDownCounter { return g.inst }

func (g MessagesCurrent) Add(ctx context.Context, incr int64, chanName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{semconv.ChanName(chanName)}
	opts := getAddOptions(base, attrs...)

	g.inst.Add(ctx, incr, opts...)

	putAddOptions(opts)
}

// ------------------------------------------------------------------------------------------------
// ~ MessagesDuration
// ------------------------------------------------------------------------------------------------

// MessagesDuration records the duration of channel message sends.
type MessagesDuration struct {
	inst metric.Float64Histogram
}

func NewMessagesDuration(m metric.Meter) (MessagesDuration, error) {
	if m == nil {
		return MessagesDuration{}, nil
	}

	h, err := m.Float64Histogram(messagesDurationName,
		metric.WithDescription(messagesDurationDesc),
		metric.WithUnit(unitSeconds),
	)

	return MessagesDuration{inst: h}, err
}

func (MessagesDuration) Name() string                    { return messagesDurationName }
func (MessagesDuration) Unit() string                    { return unitSeconds }
func (MessagesDuration) Description() string             { return messagesDurationDesc }
func (g MessagesDuration) Inst() metric.Float64Histogram { return g.inst }

func (g MessagesDuration) Record(ctx context.Context, value float64, chanName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	base := []attribute.KeyValue{semconv.ChanName(chanName)}
	opts := getRecordOptions(base, attrs...)

	g.inst.Record(ctx, value, opts...)

	putRecordOptions(opts)
}
