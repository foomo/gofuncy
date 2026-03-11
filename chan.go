package gofuncy

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

type (
	Chan[T any] struct {
		l     *slog.Logger
		level slog.Level
		name  string
		// channel
		channel chan Message[T]
		buffer  int
		// telemetry
		meter                         metric.Meter
		tracer                        trace.Tracer
		countMetric                   metric.Int64UpDownCounter
		countMetricName               string
		messagesCountMetric           metric.Int64UpDownCounter
		messagesCountMetricName       string
		messagesDurationMetric        metric.Int64Histogram
		messagesDurationMetricName    string
		messagesDurationMetricEnabled bool
		messagesAttributeEnabled      bool
		telemetryEnabled              bool
		// closing
		isClosed atomic.Bool
		closing  chan struct{}
	}
	ChanOption[T any] func(channel *Chan[T])
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func ChanWithName[T any](name string) ChanOption[T] {
	return func(o *Chan[T]) {
		o.name = name
	}
}

func ChanWithBuffer[T any](size int) ChanOption[T] {
	return func(o *Chan[T]) {
		o.buffer = size
	}
}

func ChanWithLogger[T any](l *slog.Logger) ChanOption[T] {
	return func(o *Chan[T]) {
		o.l = l
	}
}

func ChanWithLogLevel[T any](level slog.Level) ChanOption[T] {
	return func(o *Chan[T]) {
		o.level = level
	}
}

func ChanWithTelemetryEnabled[T any](enabled bool) ChanOption[T] {
	return func(o *Chan[T]) {
		o.telemetryEnabled = enabled
	}
}

func ChanWithMeter[T any](meter metric.Meter) ChanOption[T] {
	return func(o *Chan[T]) {
		o.meter = meter
	}
}

func ChanWithTracer[T any](tracer trace.Tracer) ChanOption[T] {
	return func(o *Chan[T]) {
		o.tracer = tracer
	}
}

func ChanWithCountMetricName[T any](name string) ChanOption[T] {
	return func(o *Chan[T]) {
		o.countMetricName = name
	}
}

func ChanWithMessagesCountMetricName[T any](name string) ChanOption[T] {
	return func(o *Chan[T]) {
		o.messagesCountMetricName = name
	}
}

func ChanWithMessagesDurationMetricName[T any](name string) ChanOption[T] {
	return func(o *Chan[T]) {
		o.messagesDurationMetricName = name
	}
}

func ChanWithMessagesDurationMetricEnabled[T any](enabled bool) ChanOption[T] {
	return func(o *Chan[T]) {
		o.messagesDurationMetricEnabled = enabled
	}
}

func ChanWithMessagesAttributeEnabled[T any](enabled bool) ChanOption[T] {
	return func(o *Chan[T]) {
		o.messagesAttributeEnabled = enabled
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewChan[T any](opts ...ChanOption[T]) *Chan[T] {
	inst := &Chan[T]{
		l:                          slog.New(slog.NewTextHandler(io.Discard, nil)),
		level:                      slog.LevelDebug,
		name:                       NameNoName,
		buffer:                     0,
		closing:                    make(chan struct{}),
		countMetricName:            "gofuncy.chans",
		messagesCountMetricName:    "gofuncy.messages",
		messagesDurationMetricName: "gofuncy.messages.duration",
		telemetryEnabled:           os.Getenv("GOFUNCY_TELEMETRY_ENABLED") == "true",
		messagesAttributeEnabled:   os.Getenv("GOFUNCY_MESSAGES_ATTRIBUTE_ENABLED") == "true",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(inst)
		}
	}

	inst.l = inst.l.WithGroup("gofuncy.chan").With("gofuncy_chan_name", inst.name)

	// create channel
	inst.channel = make(chan Message[T], inst.buffer)

	// create telemetry if enabled
	if inst.telemetryEnabled {
		if inst.meter == nil {
			inst.meter = otel.Meter("github.com/foomo/gofuncy")
		}

		if inst.tracer == nil {
			inst.tracer = otel.Tracer("github.com/foomo/gofuncy")
		}
	}

	if inst.meter != nil {
		if value, err := inst.meter.Int64UpDownCounter(
			inst.countMetricName,
			metric.WithDescription("Gofuncy open chan count"),
		); err != nil {
			inst.l.Warn("failed to initialize counter", "err", err)
		} else {
			inst.countMetric = value
		}

		// Initialize countMetric for this channel
		if inst.countMetric != nil {
			inst.countMetric.Add(context.Background(), 1, metric.WithAttributes(semconv.ChanName.String(inst.name)))
		}

		if value, err := inst.meter.Int64UpDownCounter(
			inst.messagesCountMetricName,
			metric.WithDescription("Gofuncy pending message count"),
		); err != nil {
			inst.l.Warn("failed to initialize counter", "err", err)
		} else {
			inst.messagesCountMetric = value
		}
	}

	if inst.meter != nil && inst.messagesDurationMetricEnabled {
		if value, err := inst.meter.Int64Histogram(
			inst.messagesDurationMetricName,
			metric.WithDescription("Gofuncy chan message send duration"),
			metric.WithUnit("ms"),
		); err != nil {
			inst.l.Warn("failed to initialize histogram", "err", err)
		} else {
			inst.messagesDurationMetric = value
		}
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (g *Chan[T]) Receive(ctx context.Context) <-chan T {
	start := time.Now()
	routineName := NameFromContext(ctx)
	l := g.l.With("gofuncy_name", routineName)

	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "gofuncy.chan.receive "+g.name, trace.WithAttributes(
			semconv.ChanCap.Int(cap(g.channel)),
			semconv.ChanSize.Int(len(g.channel)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				otelsemconv.CodeFilepath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if span.IsRecording() {
			l = l.With(
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}
		defer span.End()
	}

	out := make(chan T, 1)

	select {
	case <-ctx.Done():
		close(out)
		return out
	case msg, ok := <-g.channel:
		if !ok {
			close(out)
			return out
		}

		if g.messagesCountMetric != nil {
			g.messagesCountMetric.Add(ctx, -1, metric.WithAttributes(
				semconv.ChanName.String(g.name)),
			)
		}

		l.Debug("received messages",
			"gofuncy_sender", msg.Sender(),
			"duration", time.Since(start).Round(time.Millisecond),
		)

		out <- msg.value

		return out
	}
}

func (g *Chan[T]) Send(ctx context.Context, values ...T) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if g.isClosed.Load() {
		return ErrChanClosed
	}

	start := time.Now()
	routineName := NameFromContext(ctx)

	l := g.l.With(
		"messages", len(values),
		"gofuncy_name", routineName,
	)

	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "gofuncy.chan.send "+g.name, trace.WithAttributes(
			semconv.ChanCap.Int(cap(g.channel)),
			semconv.ChanSize.Int(len(g.channel)),
			attribute.Int("messages.total", len(values)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				otelsemconv.CodeFilepath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if span.IsRecording() {
			l = l.With(
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}
		defer span.End()
	}

	if g.messagesCountMetric != nil {
		g.messagesCountMetric.Add(ctx, int64(len(values)), metric.WithAttributes(
			semconv.ChanName.String(g.name)),
		)
	}

	message := func(ctx context.Context, span trace.Span, value T) Message[T] {
		ret := Message[T]{
			sender: routineName,
			value:  value,
		}
		if g.tracer != nil && g.messagesAttributeEnabled {
			if v, err := json.Marshal(value); err != nil {
				l.Warn("failed to marshal message value", "err", err)
			} else {
				span.AddEvent("message", trace.WithAttributes(attribute.String("value", string(v))))
			}
		}

		return ret
	}

	for _, data := range values {
		s := time.Now()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-g.closing:
			return ErrChanClosed
		case g.channel <- message(ctx, span, data):
			if g.messagesDurationMetric != nil {
				g.messagesDurationMetric.Record(ctx, time.Since(s).Milliseconds(), metric.WithAttributes(
					semconv.ChanName.String(g.name)),
				)
			}
		}
	}

	l.Debug("sent messages",
		"duration", time.Since(start).Round(time.Millisecond),
	)

	return nil
}

// Close closes the GoChannel Pub/Sub.
func (g *Chan[T]) Close() {
	if !g.isClosed.CompareAndSwap(false, true) {
		return
	}

	if g.countMetric != nil {
		g.countMetric.Add(context.Background(), -1, metric.WithAttributes(semconv.ChanName.String(g.name)))
	}

	close(g.closing)
	close(g.channel)
	g.l.Debug("closed")
}
