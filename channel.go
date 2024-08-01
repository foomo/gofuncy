package gofuncy

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type (
	Channel[T any] struct {
		l *slog.Logger
		// channel
		channel    chan *Value[T]
		bufferSize int
		// telemetry
		meter                 metric.Meter
		tracer                trace.Tracer
		counter               metric.Int64Counter
		counterName           string
		histogram             metric.Int64Histogram
		histogramName         string
		valueEventsEnabled    bool
		valueAttributeEnabled bool
		telemetryEnabled      bool
		// closing
		isClosed atomic.Bool
		closing  chan struct{}
		// closed   chan struct{}
	}
	ChannelOption[T any] func(channel *Channel[T])
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func ChannelWithLogger[T any](logger *slog.Logger) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.l = logger
	}
}

func ChannelWithBufferSize[T any](size int) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.bufferSize = size
	}
}

func ChannelWithTelemetryEnabled[T any](enabled bool) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.telemetryEnabled = enabled
	}
}

func ChannelWithMeter[T any](meter metric.Meter) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.meter = meter
	}
}

func ChannelWithCounterName[T any](name string) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.counterName = name
	}
}

func ChannelWithHistogramName(name string) Option {
	return func(o *Options) {
		o.histogramName = name
	}
}

func ChannelWithTracer[T any](tracer trace.Tracer) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.tracer = tracer
	}
}

func ChannelWithValueEventsEnabled[T any](enabled bool) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.valueEventsEnabled = enabled
	}
}

func ChannelWithValueAttributeEnabled[T any](enabled bool) ChannelOption[T] {
	return func(channel *Channel[T]) {
		channel.valueAttributeEnabled = enabled
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewChannel[T any](opts ...ChannelOption[T]) *Channel[T] {
	inst := &Channel[T]{
		l:                     slog.Default(),
		bufferSize:            0,
		counterName:           "gofuncy.channel.sent.count",
		histogramName:         "gofuncy.channel.sent.duration",
		telemetryEnabled:      os.Getenv("OTEL_ENABLED") == "true",
		valueEventsEnabled:    os.Getenv("GOFUNCY_CHANNEL_VALUE_EVENTS_ENABLED") == "true",
		valueAttributeEnabled: os.Getenv("GOFUNCY_CHANNEL_VALUE_ATTRIBUTE_ENABLED") == "true",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(inst)
		}
	}

	// create channel
	inst.channel = make(chan *Value[T], inst.bufferSize)

	// create telemetry if enabled
	if inst.telemetryEnabled {
		if inst.meter == nil {
			inst.meter = otel.Meter("gofuncy")
		}
		if value, err := inst.meter.Int64Counter(inst.counterName); err != nil {
			inst.l.Error("failed to initialize counter", "error", err)
		} else {
			inst.counter = value
		}
		if value, err := inst.meter.Int64Histogram(inst.histogramName); err != nil {
			inst.l.Error("failed to initialize histogram", "error", err)
		} else {
			inst.histogram = value
		}
		if inst.tracer == nil {
			inst.tracer = otel.Tracer("gofuncy")
		}
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (g *Channel[T]) Receive() <-chan *Value[T] {
	return g.channel
}

func (g *Channel[T]) Send(ctx context.Context, values ...T) error {
	if g.isClosed.Load() {
		return ErrChannelClosed
	}
	if g.histogram != nil {
		start := time.Now()
		defer func() {
			g.histogram.Record(ctx, time.Since(start).Milliseconds())
		}()
	}

	var span trace.Span
	if g.tracer != nil {
		ctx, span = g.tracer.Start(ctx, "Send", trace.WithAttributes(
			attribute.Int("num", len(values)),
			attribute.Int("chan_cap", cap(g.channel)),
			attribute.Int("chan_size", len(g.channel)),
		))
	}
	defer span.End()

	newValue := func(ctx context.Context, span trace.Span, data T) *Value[T] {
		ret := &Value[T]{
			ctx:  injectSenderIntoContext(ctx, RoutineFromContext(ctx)),
			Data: data,
		}
		if g.counter != nil {
			g.counter.Add(ctx, 1)
		}
		if g.tracer != nil {
			if g.valueEventsEnabled {
				var attrs []attribute.KeyValue
				if g.valueAttributeEnabled {
					if value, err := json.Marshal(data); err == nil {
						attrs = append(attrs, attribute.String("value", string(value)))
					}
				}
				span.AddEvent("value", trace.WithAttributes(attrs...))
			}
		}
		return ret
	}

	for _, data := range values {
		select {
		case <-g.closing:
			return ErrChannelClosed
		default:
		}

		select {
		case <-g.closing:
			return ErrChannelClosed
		case g.channel <- newValue(ctx, span, data):
		}
	}

	return nil
}

// Close closes the GoChannel Pub/Sub.
func (g *Channel[T]) Close() {
	if g.isClosed.Load() {
		return
	}
	g.isClosed.Store(true)
	g.closing <- struct{}{}
	close(g.channel)
}
