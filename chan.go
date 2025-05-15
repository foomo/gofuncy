package gofuncy

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Chan[T any] struct {
		l     *zap.Logger
		level zapcore.Level
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

func ChanWithLogger[T any](l *zap.Logger) ChanOption[T] {
	return func(o *Chan[T]) {
		o.l = l
	}
}

func ChanWithLogLevel[T any](level zapcore.Level) ChanOption[T] {
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
		l:                          zap.NewNop(),
		level:                      zapcore.DebugLevel,
		name:                       NameNoName,
		buffer:                     0,
		closing:                    make(chan struct{}, 1),
		countMetricName:            "gofuncy.chans",
		messagesCountMetricName:    "gofuncy.messages",
		messagesDurationMetricName: "gofuncy.messages.duration",
		telemetryEnabled:           os.Getenv("OTEL_ENABLED") == "true",
		messagesAttributeEnabled:   os.Getenv("GOFUNCY_MESSAGES_ATTRIBUTE_ENABLED") == "true",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(inst)
		}
	}

	inst.l = inst.l.Named("gofuncy.chan").With(
		zap.String("gofuncy_chan_name", inst.name),
	)

	// create channel
	if inst.buffer == 0 {
		inst.channel = make(chan Message[T])
	} else {
		inst.channel = make(chan Message[T], inst.buffer)
	}

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
			inst.l.Warn("failed to initialize counter", zap.Error(err))
		} else {
			inst.countMetric = value
		}

		if value, err := inst.meter.Int64UpDownCounter(
			inst.messagesCountMetricName,
			metric.WithDescription("Gofuncy pending message count"),
		); err != nil {
			inst.l.Warn("failed to initialize counter", zap.Error(err))
		} else {
			inst.messagesCountMetric = value
		}
	}

	if inst.meter != nil && inst.messagesDurationMetricEnabled {
		if value, err := inst.meter.Int64Histogram(
			inst.messagesDurationMetricName,
			metric.WithDescription("Gofuncy chan message send duration"),
		); err != nil {
			inst.l.Warn("failed to initialize histogram", zap.Error(err))
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
	l := g.l.With(
		zap.String("gofuncy_name", routineName),
	)
	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "GOFUNCY receive."+g.name, trace.WithAttributes(
			attribute.Int("chan.cap", cap(g.channel)),
			attribute.Int("chan.size", len(g.channel)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				semconv.CodeFilepath(file),
				semconv.CodeLineNumber(line),
				semconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if span.IsRecording() {
			l = l.With(
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.String("span_id", span.SpanContext().SpanID().String()),
			)
		}
		defer span.End()
	}
	out := make(chan T, 1)
	for {
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
					attribute.String("chan_name", g.name)),
				)
			}
			l.Debug("received messages",
				zap.String("gofuncy_sender", msg.Sender()),
				zap.Duration("duration", time.Since(start).Round(time.Millisecond)),
			)
			out <- msg.value
			return out
		}
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
		zap.Int("messages", len(values)),
		zap.String("gofuncy_name", routineName),
	)

	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "GOFUNCY send."+g.name, trace.WithAttributes(
			attribute.Int("chan.cap", cap(g.channel)),
			attribute.Int("chan.size", len(g.channel)),
			attribute.Int("messages.total", len(values)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				semconv.CodeFilepath(file),
				semconv.CodeLineNumber(line),
				semconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if span.IsRecording() {
			l = l.With(
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.String("span_id", span.SpanContext().SpanID().String()),
			)
		}
		defer span.End()
	}

	if g.messagesCountMetric != nil {
		g.messagesCountMetric.Add(ctx, int64(len(values)), metric.WithAttributes(
			attribute.String("chan_name", g.name)),
		)
	}

	message := func(ctx context.Context, span trace.Span, value T) Message[T] {
		ret := Message[T]{
			sender: routineName,
			value:  value,
		}
		if g.tracer != nil && g.messagesAttributeEnabled {
			if v, err := json.Marshal(value); err != nil {
				l.Warn("failed to marshal message value", zap.Error(err))
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
				g.messagesDurationMetric.Record(ctx, time.Since(s).Milliseconds())
			}
		}
	}

	l.Debug("sent messages",
		zap.Duration("duration", time.Since(start).Round(time.Millisecond)),
	)

	return nil
}

// Close closes the GoChannel Pub/Sub.
func (g *Chan[T]) Close() {
	if g.isClosed.Load() {
		return
	}
	g.isClosed.Store(true)
	g.closing <- struct{}{}
	close(g.channel)
	g.l.Debug("closed")
}
