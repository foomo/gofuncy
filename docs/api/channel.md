---
prev:
  text: Options
  link: /api/options
next:
  text: Basic Examples
  link: /examples/basic

---

# Channel

A generic, telemetry-aware channel wrapper. Provides observable `Send` and `Receive` operations with opt-in/out metrics and tracing.

Lives in the `github.com/foomo/gofuncy/channel` subpackage.

## Signature

```go
func New[T any](name string, opts ...Option[T]) *Channel[T]
```

### Type Parameters

| Parameter | Description |
|-----------|-------------|
| `T` | The type of values sent through the channel. |

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | `string` | Name for the channel. Used in telemetry metrics and traces. |
| `opts` | `...Option[T]` | Functional options to configure the channel. |

## Options

### Defaults

| Feature | Default | Option |
|---------|---------|--------|
| Buffer size | `0` (unbuffered) | `WithBuffer[T](size)` |
| Logger | `slog.Default()` | `WithLogger[T](l)` |
| Chans counter (`gofuncy.chans.current`) | **on** | `WithoutChansCounter[T]()` |
| Messages counter (`gofuncy.messages.current`) | **on** | `WithoutMessagesCounter[T]()` |
| Duration histogram (`gofuncy.messages.duration.seconds`) | off | `WithDurationHistogram[T]()` |
| Tracing | off | `WithTracing[T]()` |
| Meter provider | OTel global | `WithMeterProvider[T](mp)` |
| Tracer provider | OTel global | `WithTracerProvider[T](tp)` |

::: tip
Counters are cheap and enabled by default. Duration histogram and tracing are opt-in because they add overhead on every `Send` call. This matches the convention used by `Go`, `Do`, `Start`, and `Group`.
:::

## Behavior

1. `New` creates the underlying Go channel and initializes telemetry instruments based on the enabled options.
2. If the chans counter is enabled, the channel increments `gofuncy.chans.current` on creation and decrements it on `Close`.
3. `Send` writes values to the channel one at a time. For each value:
   - If the context is cancelled, returns the context error immediately.
   - If the channel is closed, returns `channel.ErrClosed`.
   - If the messages counter is enabled, increments `gofuncy.messages.current`.
   - If the duration histogram is enabled, records the time spent waiting for the channel to accept the value (backpressure detection).
   - If tracing is enabled, adds a span event for each sent value.
4. `Receive` returns the raw underlying `<-chan T`. This is zero-allocation and works with `range`.
5. `Close` is idempotent — safe to call multiple times. It broadcasts to all blocked senders, then closes the underlying channel.

## Methods

### Send

```go
func (c *Channel[T]) Send(ctx context.Context, values ...T) error
```

Sends one or more values into the channel. Returns `channel.ErrClosed` if the channel has been closed, or the context error if the context is cancelled while waiting.

### Receive

```go
func (c *Channel[T]) Receive() <-chan T
```

Returns a read-only view of the underlying channel. Use with `range` or `select`.

### Close

```go
func (c *Channel[T]) Close()
```

Closes the channel. Idempotent — subsequent calls are no-ops. Unblocks any goroutines waiting in `Send`.

### Len / Cap / Name

```go
func (c *Channel[T]) Len() int
func (c *Channel[T]) Cap() int
func (c *Channel[T]) Name() string
```

Return the current number of buffered values, the buffer capacity, and the channel name.

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/foomo/gofuncy/channel"
)

func main() {
	ctx := context.Background()

	// Create a buffered channel with default telemetry (counters on)
	ch := channel.New[string]("events", channel.WithBuffer[string](10))

	// Producer
	gofuncy.Go(ctx, "producer", func(ctx context.Context) error {
		defer ch.Close()

		for i := range 5 {
			if err := ch.Send(ctx, fmt.Sprintf("event-%d", i)); err != nil {
				return err
			}
		}

		return nil
	})

	// Consumer
	for msg := range ch.Receive() {
		fmt.Println(msg)
	}

	time.Sleep(100 * time.Millisecond)
}
```

## Telemetry Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `gofuncy.chans.current` | UpDownCounter | Number of open channels. Attributes: `gofuncy.chan.name`, `gofuncy.chan.cap`. |
| `gofuncy.messages.current` | UpDownCounter | Number of in-flight (buffered) messages. Attributes: `gofuncy.chan.name`. |
| `gofuncy.messages.duration.seconds` | Histogram | Time spent waiting for the channel to accept a value. High values indicate backpressure. Attributes: `gofuncy.chan.name`, `gofuncy.chan.cap`, `gofuncy.chan.size`. |

::: warning
`Receive()` returns the raw Go channel, so the messages counter is incremented on `Send` but not decremented on receive. Use the counter to detect stuck or filling channels — if it only goes up, nothing is consuming.
:::
