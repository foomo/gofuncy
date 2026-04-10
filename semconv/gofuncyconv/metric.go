package gofuncyconv

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/foomo/gofuncy/semconv"
)

// ------------------------------------------------------------------------------------------------
// ~ Constants
// ------------------------------------------------------------------------------------------------

const (
	goroutinesStartedName = "gofuncy.goroutines.started"
	goroutinesStartedDesc = "Total number of goroutines started"

	goroutinesErrorsName = "gofuncy.goroutines.errors"
	goroutinesErrorsDesc = "Total number of goroutine errors"

	goroutinesRetriesName  = "gofuncy.goroutines.retries"
	goroutinesRetriesDesc  = "Total number of retry attempts"
	goroutinesRejectedName = "gofuncy.goroutines.circuitbreaker.rejected"
	goroutinesRejectedDesc = "Total number of requests rejected by a circuit breaker"

	goroutinesStalledName = "gofuncy.goroutines.stalled"
	goroutinesStalledDesc = "Total number of goroutines that exceeded their stall threshold"

	goroutinesActiveName = "gofuncy.goroutines.active"
	goroutinesActiveDesc = "Number of currently active goroutines"

	goroutinesDurationName = "gofuncy.goroutines.duration.seconds"
	goroutinesDurationDesc = "Duration of goroutine execution"

	groupsDurationName = "gofuncy.groups.duration.seconds"
	groupsDurationDesc = "Gofuncy group/map duration histogram"

	chansCurrentName = "gofuncy.chans.current"
	chansCurrentDesc = "Gofuncy open chan up/down count"

	messagesSentName = "gofuncy.messages.sent"
	messagesSentDesc = "Total number of messages sent"

	messagesDurationName = "gofuncy.messages.duration.seconds"
	messagesDurationDesc = "Gofuncy chan message send duration"

	unitGoroutine = "{goroutine}"
	unitSeconds   = "s"
	unitChan      = "{chan}"
	unitMessage   = "{message}"
)

// default histogram bucket boundaries for goroutine/group durations
var durationBuckets = metric.WithExplicitBucketBoundaries(
	0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 300.0, 600.0,
)

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesStarted
// ------------------------------------------------------------------------------------------------

// GoroutinesStarted counts the total number of goroutines started.
type GoroutinesStarted struct {
	inst metric.Int64Counter
}

// NewGoroutinesStarted creates a new goroutines started counter.
func NewGoroutinesStarted(m metric.Meter) (GoroutinesStarted, error) {
	if m == nil {
		return GoroutinesStarted{}, nil
	}

	c, err := m.Int64Counter(goroutinesStartedName,
		metric.WithDescription(goroutinesStartedDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesStarted{inst: c}, err
}

func (GoroutinesStarted) Name() string                { return goroutinesStartedName }
func (GoroutinesStarted) Unit() string                { return unitGoroutine }
func (GoroutinesStarted) Description() string         { return goroutinesStartedDesc }
func (g GoroutinesStarted) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesStarted) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesErrors
// ------------------------------------------------------------------------------------------------

// GoroutinesErrors counts the total number of goroutine errors.
type GoroutinesErrors struct {
	inst metric.Int64Counter
}

// NewGoroutinesErrors creates a new goroutine errors counter.
func NewGoroutinesErrors(m metric.Meter) (GoroutinesErrors, error) {
	if m == nil {
		return GoroutinesErrors{}, nil
	}

	c, err := m.Int64Counter(goroutinesErrorsName,
		metric.WithDescription(goroutinesErrorsDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesErrors{inst: c}, err
}

func (GoroutinesErrors) Name() string                { return goroutinesErrorsName }
func (GoroutinesErrors) Unit() string                { return unitGoroutine }
func (GoroutinesErrors) Description() string         { return goroutinesErrorsDesc }
func (g GoroutinesErrors) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesErrors) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesRetries
// ------------------------------------------------------------------------------------------------

// GoroutinesRetries counts the total number of retry attempts.
type GoroutinesRetries struct {
	inst metric.Int64Counter
}

// NewGoroutinesRetries creates a new retry attempts counter.
func NewGoroutinesRetries(m metric.Meter) (GoroutinesRetries, error) {
	if m == nil {
		return GoroutinesRetries{}, nil
	}

	c, err := m.Int64Counter(goroutinesRetriesName,
		metric.WithDescription(goroutinesRetriesDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesRetries{inst: c}, err
}

func (GoroutinesRetries) Name() string                { return goroutinesRetriesName }
func (GoroutinesRetries) Unit() string                { return unitGoroutine }
func (GoroutinesRetries) Description() string         { return goroutinesRetriesDesc }
func (g GoroutinesRetries) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesRetries) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesRejected
// ------------------------------------------------------------------------------------------------

// GoroutinesRejected counts requests rejected by a circuit breaker.
type GoroutinesRejected struct {
	inst metric.Int64Counter
}

// NewGoroutinesRejected creates a new circuit breaker rejections counter.
func NewGoroutinesRejected(m metric.Meter) (GoroutinesRejected, error) {
	if m == nil {
		return GoroutinesRejected{}, nil
	}

	c, err := m.Int64Counter(goroutinesRejectedName,
		metric.WithDescription(goroutinesRejectedDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesRejected{inst: c}, err
}

func (GoroutinesRejected) Name() string                { return goroutinesRejectedName }
func (GoroutinesRejected) Unit() string                { return unitGoroutine }
func (GoroutinesRejected) Description() string         { return goroutinesRejectedDesc }
func (g GoroutinesRejected) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesRejected) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesStalled
// ------------------------------------------------------------------------------------------------

// GoroutinesStalled counts the total number of goroutines that exceeded their stall threshold.
type GoroutinesStalled struct {
	inst metric.Int64Counter
}

// NewGoroutinesStalled creates a new stalled goroutines counter.
func NewGoroutinesStalled(m metric.Meter) (GoroutinesStalled, error) {
	if m == nil {
		return GoroutinesStalled{}, nil
	}

	c, err := m.Int64Counter(goroutinesStalledName,
		metric.WithDescription(goroutinesStalledDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesStalled{inst: c}, err
}

func (GoroutinesStalled) Name() string                { return goroutinesStalledName }
func (GoroutinesStalled) Unit() string                { return unitGoroutine }
func (GoroutinesStalled) Description() string         { return goroutinesStalledDesc }
func (g GoroutinesStalled) Inst() metric.Int64Counter { return g.inst }

func (g GoroutinesStalled) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesActive
// ------------------------------------------------------------------------------------------------

// GoroutinesActive tracks the number of currently active goroutines.
type GoroutinesActive struct {
	inst metric.Int64UpDownCounter
}

// NewGoroutinesActive creates a new active goroutines up-down counter.
func NewGoroutinesActive(m metric.Meter) (GoroutinesActive, error) {
	if m == nil {
		return GoroutinesActive{}, nil
	}

	c, err := m.Int64UpDownCounter(goroutinesActiveName,
		metric.WithDescription(goroutinesActiveDesc),
		metric.WithUnit(unitGoroutine),
	)

	return GoroutinesActive{inst: c}, err
}

func (GoroutinesActive) Name() string                      { return goroutinesActiveName }
func (GoroutinesActive) Unit() string                      { return unitGoroutine }
func (GoroutinesActive) Description() string               { return goroutinesActiveDesc }
func (g GoroutinesActive) Inst() metric.Int64UpDownCounter { return g.inst }

func (g GoroutinesActive) Add(ctx context.Context, incr int64, routineName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.RoutineName(routineName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.RoutineName(routineName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ GoroutinesDuration
// ------------------------------------------------------------------------------------------------

// GoroutinesDuration records the duration of goroutine execution.
type GoroutinesDuration struct {
	inst metric.Float64Histogram
}

// NewGoroutinesDuration creates a new goroutine duration histogram.
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

	if len(attrs) == 0 {
		g.inst.Record(ctx, value, metric.WithAttributes(
			semconv.RoutineName(routineName),
			semconv.Error(hasError),
		))

		return
	}

	g.inst.Record(ctx, value, metric.WithAttributes(append(attrs,
		semconv.RoutineName(routineName),
		semconv.Error(hasError),
	)...))
}

// ------------------------------------------------------------------------------------------------
// ~ GroupsDuration
// ------------------------------------------------------------------------------------------------

// GroupsDuration records the duration of group/map operations.
type GroupsDuration struct {
	inst metric.Float64Histogram
}

// NewGroupsDuration creates a new group duration histogram.
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

func (g GroupsDuration) Record(ctx context.Context, value float64, routineName string, hasError bool, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Record(ctx, value, metric.WithAttributes(
			semconv.RoutineName(routineName),
			semconv.Error(hasError),
		))

		return
	}

	g.inst.Record(ctx, value, metric.WithAttributes(append(attrs,
		semconv.RoutineName(routineName),
		semconv.Error(hasError),
	)...))
}

// ------------------------------------------------------------------------------------------------
// ~ ChansCurrent
// ------------------------------------------------------------------------------------------------

// ChansCurrent tracks the current number of open channels.
type ChansCurrent struct {
	inst metric.Int64UpDownCounter
}

// NewChansCurrent creates a new open channels up-down counter.
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

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.ChanName(chanName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.ChanName(chanName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ MessagesSent
// ------------------------------------------------------------------------------------------------

// MessagesSent counts the total number of messages sent.
type MessagesSent struct {
	inst metric.Int64Counter
}

// NewMessagesSent creates a new messages sent counter.
func NewMessagesSent(m metric.Meter) (MessagesSent, error) {
	if m == nil {
		return MessagesSent{}, nil
	}

	c, err := m.Int64Counter(messagesSentName,
		metric.WithDescription(messagesSentDesc),
		metric.WithUnit(unitMessage),
	)

	return MessagesSent{inst: c}, err
}

func (MessagesSent) Name() string                { return messagesSentName }
func (MessagesSent) Unit() string                { return unitMessage }
func (MessagesSent) Description() string         { return messagesSentDesc }
func (g MessagesSent) Inst() metric.Int64Counter { return g.inst }

func (g MessagesSent) Add(ctx context.Context, incr int64, chanName string, attrs ...attribute.KeyValue) {
	if g.inst == nil {
		return
	}

	if len(attrs) == 0 {
		g.inst.Add(ctx, incr, metric.WithAttributes(semconv.ChanName(chanName)))
		return
	}

	g.inst.Add(ctx, incr, metric.WithAttributes(append(attrs, semconv.ChanName(chanName))...))
}

// ------------------------------------------------------------------------------------------------
// ~ MessagesDuration
// ------------------------------------------------------------------------------------------------

// MessagesDuration records the duration of channel message sends.
type MessagesDuration struct {
	inst metric.Float64Histogram
}

// NewMessagesDuration creates a new message send duration histogram.
func NewMessagesDuration(m metric.Meter) (MessagesDuration, error) {
	if m == nil {
		return MessagesDuration{}, nil
	}

	h, err := m.Float64Histogram(messagesDurationName,
		metric.WithDescription(messagesDurationDesc),
		metric.WithUnit(unitSeconds),
		durationBuckets,
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

	if len(attrs) == 0 {
		g.inst.Record(ctx, value, metric.WithAttributes(semconv.ChanName(chanName)))
		return
	}

	g.inst.Record(ctx, value, metric.WithAttributes(append(attrs, semconv.ChanName(chanName))...))
}
