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
	Channel[T any] struct {
		name string
		ch   chan T
		l    *slog.Logger

		// feature flags — all default true
		chansCounter      bool
		messagesCounter   bool
		durationHistogram bool
		tracing           bool

		// pre-resolved instruments
		chansCurrent     gofuncyconv.ChansCurrent
		messagesCurrent  gofuncyconv.MessagesCurrent
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
	Option[T any] func(*Channel[T])
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func WithBuffer[T any](size int) Option[T] {
	return func(c *Channel[T]) {
		c.bufferSize = size
	}
}

func WithLogger[T any](l *slog.Logger) Option[T] {
	return func(c *Channel[T]) {
		c.l = l
	}
}

func WithMeterProvider[T any](mp metric.MeterProvider) Option[T] {
	return func(c *Channel[T]) {
		c.meterProvider = mp
	}
}

func WithTracerProvider[T any](tp trace.TracerProvider) Option[T] {
	return func(c *Channel[T]) {
		c.tracerProvider = tp
	}
}

func WithoutChansCounter[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.chansCounter = false
	}
}

func WithoutMessagesCounter[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.messagesCounter = false
	}
}

func WithDurationHistogram[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.durationHistogram = true
	}
}

func WithTracing[T any]() Option[T] {
	return func(c *Channel[T]) {
		c.tracing = true
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func New[T any](name string, opts ...Option[T]) *Channel[T] {
	c := &Channel[T]{
		name:            name,
		l:               slog.Default(),
		chansCounter:    true,
		messagesCounter: true,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	c.ch = make(chan T, c.bufferSize)
	c.closing = make(chan struct{})

	if c.chansCounter || c.messagesCounter || c.durationHistogram {
		m := c.meter()

		if c.chansCounter {
			if v, err := gofuncyconv.NewChansCurrent(m); err != nil {
				c.l.Error("failed to create chans current metric", slog.String("error", err.Error()))
			} else {
				c.chansCurrent = v
			}
		}

		if c.messagesCounter {
			if v, err := gofuncyconv.NewMessagesCurrent(m); err != nil {
				c.l.Error("failed to create messages current metric", slog.String("error", err.Error()))
			} else {
				c.messagesCurrent = v
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

		if c.messagesCounter {
			c.messagesCurrent.Add(ctx, 1, c.name)
		}

		if c.tracing && span != nil {
			span.AddEvent("sent")
		}
	}

	return nil
}

func (c *Channel[T]) Receive() <-chan T {
	return c.ch
}

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

func (c *Channel[T]) Len() int {
	return len(c.ch)
}

func (c *Channel[T]) Cap() int {
	return cap(c.ch)
}

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
