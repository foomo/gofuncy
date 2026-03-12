package gofuncy

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy/semconv"
)

type (
	Chan[T any] struct {
		l    *slog.Logger
		name string
		// channel
		channel chan Message[T]
		buffer  int
		// telemetry
		tracingEnabled                bool
		counterMetricEnabled          bool
		messagesDurationMetricEnabled bool
		messagesAttributeEnabled      bool
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

func ChanWithTracing[T any]() ChanOption[T] {
	return func(o *Chan[T]) {
		o.tracingEnabled = true
	}
}

func ChanWithCounterMetric[T any]() ChanOption[T] {
	return func(o *Chan[T]) {
		o.counterMetricEnabled = true
	}
}

func ChanWithMessagesDurationMetric[T any]() ChanOption[T] {
	return func(o *Chan[T]) {
		o.messagesDurationMetricEnabled = true
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
		l:                        nil,
		name:                     NameNoName,
		buffer:                   0,
		closing:                  make(chan struct{}),
		messagesAttributeEnabled: defaultMessagesAttributeEnabled,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(inst)
		}
	}

	// Only chain logger if one was provided (avoid allocation)
	if inst.l != nil {
		inst.l = inst.l.WithGroup("gofuncy.chan").With("gofuncy_chan_name", inst.name)
	}

	// create channel
	inst.channel = make(chan Message[T], inst.buffer)

	// create counter metric if enabled
	if inst.counterMetricEnabled {
		chansCounter().Add(context.Background(), 1, metric.WithAttributes(semconv.ChanName(inst.name)))
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (g *Chan[T]) Receive(ctx context.Context) <-chan T {
	var (
		l     *slog.Logger
		span  trace.Span
		start time.Time
	)
	if g.l != nil {
		start = time.Now()
		l = g.l.With("gofuncy_name", NameFromContext(ctx))
	}

	if g.tracingEnabled {
		ctx, span = tracer.Start(ctx, "gofuncy.chan.receive "+g.name, trace.WithAttributes(
			semconv.ChanCap(cap(g.channel)),
			semconv.ChanSize(len(g.channel)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				otelsemconv.CodeFilePath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if l != nil && span.IsRecording() {
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

		if g.counterMetricEnabled {
			messagesCounter().Add(ctx, -1, metric.WithAttributes(
				semconv.ChanName(g.name)),
			)
		}

		if l != nil {
			l.Debug("received messages",
				"gofuncy_sender", msg.Sender(),
				"duration", time.Since(start).Round(time.Millisecond),
			)
		}

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

	routineName := NameFromContext(ctx)

	var (
		l     *slog.Logger
		start time.Time
	)
	if g.l != nil {
		start = time.Now()
		l = g.l.With(
			"messages", len(values),
			"gofuncy_name", routineName,
		)
	}

	var span trace.Span
	if g.tracingEnabled {
		ctx, span = tracer.Start(ctx, "gofuncy.chan.send "+g.name, trace.WithAttributes(
			semconv.ChanCap(cap(g.channel)),
			semconv.ChanSize(len(g.channel)),
			attribute.Int("messages.total", len(values)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				otelsemconv.CodeFilePath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if l != nil && span.IsRecording() {
			l = l.With(
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}
		defer span.End()
	}

	if g.counterMetricEnabled {
		messagesCounter().Add(ctx, int64(len(values)), metric.WithAttributes(
			semconv.ChanName(g.name)),
		)
	}

	for _, data := range values {
		s := time.Now()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-g.closing:
			return ErrChanClosed
		case g.channel <- g.newMessage(span, l, routineName, data):
			if g.messagesDurationMetricEnabled {
				messagesDurationHistogram().Record(ctx, time.Since(s).Milliseconds(), metric.WithAttributes(
					semconv.ChanName(g.name)),
				)
			}
		}
	}

	if l != nil {
		l.Debug("sent messages",
			"duration", time.Since(start).Round(time.Millisecond),
		)
	}

	return nil
}

// ReceiveValue receives a single value without allocating a channel (OPT 5: direct value return)
func (g *Chan[T]) ReceiveValue(ctx context.Context) (T, bool) {
	var (
		l     *slog.Logger
		span  trace.Span
		start time.Time
	)
	if g.l != nil {
		start = time.Now()
		l = g.l.With("gofuncy_name", NameFromContext(ctx))
	}

	if g.tracingEnabled {
		ctx, span = tracer.Start(ctx, "gofuncy.chan.receive "+g.name, trace.WithAttributes(
			semconv.ChanCap(cap(g.channel)),
			semconv.ChanSize(len(g.channel)),
		))
		// add caller
		if pc, file, line, ok := runtime.Caller(1); ok {
			span.SetAttributes(
				otelsemconv.CodeFilePath(file),
				otelsemconv.CodeLineNumber(line),
				otelsemconv.CodeFunctionName(runtime.FuncForPC(pc).Name()),
			)
		}
		// enrich logger
		if l != nil && span.IsRecording() {
			l = l.With(
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}
		defer span.End()
	}

	select {
	case <-ctx.Done():
		var zero T
		return zero, false
	case msg, ok := <-g.channel:
		if !ok {
			var zero T
			return zero, false
		}

		if g.counterMetricEnabled {
			messagesCounter().Add(ctx, -1, metric.WithAttributes(
				semconv.ChanName(g.name)),
			)
		}

		if l != nil {
			l.Debug("received messages",
				"gofuncy_sender", msg.Sender(),
				"duration", time.Since(start).Round(time.Millisecond),
			)
		}

		return msg.value, true
	}
}

// newMessage creates a Message with optional tracing support (OPT 4: method instead of closure)
func (g *Chan[T]) newMessage(span trace.Span, l *slog.Logger, routineName string, value T) Message[T] {
	ret := Message[T]{
		sender: routineName,
		value:  value,
	}
	if g.tracingEnabled && g.messagesAttributeEnabled {
		if v, err := json.Marshal(value); err != nil {
			if l != nil {
				l.Warn("failed to marshal message value", "err", err)
			}
		} else {
			span.AddEvent("message", trace.WithAttributes(attribute.String("value", string(v))))
		}
	}

	return ret
}

// Close closes the GoChannel Pub/Sub.
func (g *Chan[T]) Close() {
	if !g.isClosed.CompareAndSwap(false, true) {
		return
	}

	if g.counterMetricEnabled {
		chansCounter().Add(context.Background(), -1, metric.WithAttributes(semconv.ChanName(g.name)))
	}

	close(g.closing)
	close(g.channel)

	if g.l != nil {
		g.l.Debug("closed")
	}
}
