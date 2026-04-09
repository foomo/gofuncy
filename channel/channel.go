package channel

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/gofuncy"
	"github.com/foomo/gofuncy/semconv"
	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

// ErrClosed is returned when sending on a closed channel.
var ErrClosed = errors.New("channel is closed")

// ------------------------------------------------------------------------------------------------
// ~ Types
// ------------------------------------------------------------------------------------------------

type (
	// Channel is a generic, observable channel with optional telemetry.
	Channel[T any] struct {
		name string
		ch   chan T
		l    *slog.Logger

		// feature flags — all default true
		chansCounter        bool
		messagesSentCounter bool
		durationHistogram   bool
		tracing             bool

		// pre-resolved instruments
		chansCurrent     gofuncyconv.ChansCurrent
		messagesSent     gofuncyconv.MessagesSent
		messagesDuration gofuncyconv.MessagesDuration
		tracer           trace.Tracer

		// telemetry providers
		meterProvider  metric.MeterProvider
		tracerProvider trace.TracerProvider

		// safe close
		mu       sync.RWMutex
		isClosed atomic.Bool
		closing  chan struct{}

		// config
		bufferSize int
	}
	// Option configures a Channel during construction.
	Option[T any] func(*Channel[T])
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

// WithBuffer sets the channel buffer size.
func WithBuffer[T any](size int) Option[T] {
	return func(c *Channel[T]) {
		c.bufferSize = size
	}
}

// WithLogger sets the logger for telemetry errors.
func WithLogger[T any](l *slog.Logger) Option[T] {
	return func(c *Channel[T]) {
		c.l = l
	}
}

// WithMeterProvider sets a custom OTel meter provider.
func WithMeterProvider[T any](mp metric.MeterProvider) Option[T] {
	return func(c *Channel[T]) {
		c.meterProvider = mp
	}
}

// WithTracerProvider sets a custom OTel tracer provider.
func WithTracerProvider[T any](tp trace.TracerProvider) Option[T] {
	return func(c *Channel[T]) {
		c.tracerProvider = tp
	}
}

// WithoutChansCounter disables the open channels counter metric.
func WithoutChansCounter[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.chansCounter = false
	}
}

// WithoutMessagesSentCounter disables the messages sent counter metric.
func WithoutMessagesSentCounter[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.messagesSentCounter = false
	}
}

// WithDurationHistogram enables the message send duration histogram.
func WithDurationHistogram[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.durationHistogram = true
	}
}

// WithTracing enables OpenTelemetry tracing for send operations.
func WithTracing[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.tracing = true
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

// New creates a new Channel with the given name and options.
func New[T any](name string, opts ...Option[T]) *Channel[T] {
	c := &Channel[T]{
		name:                name,
		l:                   slog.Default(),
		chansCounter:        true,
		messagesSentCounter: true,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	c.ch = make(chan T, c.bufferSize)
	c.closing = make(chan struct{})

	if c.chansCounter || c.messagesSentCounter || c.durationHistogram {
		m := c.meter()

		if c.chansCounter {
			if v, err := gofuncyconv.NewChansCurrent(m); err != nil {
				c.l.Error("failed to create chans current metric", slog.String("error", err.Error()))
			} else {
				c.chansCurrent = v
			}
		}

		if c.messagesSentCounter {
			if v, err := gofuncyconv.NewMessagesSent(m); err != nil {
				c.l.Error("failed to create messages sent metric", slog.String("error", err.Error()))
			} else {
				c.messagesSent = v
			}
		}

		if c.durationHistogram {
			if v, err := gofuncyconv.NewMessagesDuration(m); err != nil {
				c.l.Error("failed to create messages duration metric", slog.String("error", err.Error()))
			} else {
				c.messagesDuration = v
			}
		}
	}

	if c.chansCounter {
		c.chansCurrent.Add(context.Background(), 1, c.name, semconv.ChanCap(cap(c.ch)))
	}

	if c.tracing {
		c.tracer = c.tracerFn()
	}

	return c
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

// Send sends one or more values into the channel. Returns ErrClosed if the
// channel has been closed, or the context error if the context is cancelled.
func (c *Channel[T]) Send(ctx context.Context, values ...T) error {
	if c.isClosed.Load() {
		return ErrClosed
	}

	var span trace.Span
	if c.tracing {
		ctx, span = c.tracer.Start(ctx, "gofuncy.channel.send "+c.name,
			trace.WithAttributes(
				semconv.ChanName(c.name),
				semconv.ChanCap(cap(c.ch)),
				semconv.ChanSize(len(c.ch)),
			),
		)
		defer span.End()
	}

	for _, value := range values {
		if c.durationHistogram {
			start := time.Now()

			if err := c.sendOne(ctx, value); err != nil {
				return err
			}

			c.messagesDuration.Record(ctx, time.Since(start).Seconds(), c.name)
		} else {
			if err := c.sendOne(ctx, value); err != nil {
				return err
			}
		}

		if c.messagesSentCounter {
			c.messagesSent.Add(ctx, 1, c.name)
		}

		if c.tracing && span != nil {
			span.AddEvent("sent")
		}
	}

	return nil
}

// Receive returns the underlying receive-only channel.
func (c *Channel[T]) Receive() <-chan T {
	return c.ch
}

// Close closes the channel. It is safe to call multiple times.
func (c *Channel[T]) Close() {
	if !c.isClosed.CompareAndSwap(false, true) {
		return
	}

	close(c.closing)
	c.mu.Lock()
	close(c.ch)
	c.mu.Unlock()

	if c.chansCounter {
		c.chansCurrent.Add(context.Background(), -1, c.name, semconv.ChanCap(cap(c.ch)))
	}
}

// Len returns the number of elements currently in the channel buffer.
func (c *Channel[T]) Len() int {
	return len(c.ch)
}

// Cap returns the channel buffer capacity.
func (c *Channel[T]) Cap() int {
	return cap(c.ch)
}

// Name returns the channel name.
func (c *Channel[T]) Name() string {
	return c.name
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (c *Channel[T]) sendOne(ctx context.Context, value T) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closing:
		return ErrClosed
	case c.ch <- value:
		return nil
	}
}

func (c *Channel[T]) meter() metric.Meter {
	mp := c.meterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	return mp.Meter(gofuncy.ScopeName, metric.WithSchemaURL(otelsemconv.SchemaURL))
}

func (c *Channel[T]) tracerFn() trace.Tracer {
	tp := c.tracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(gofuncy.ScopeName)
}
