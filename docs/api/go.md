---
prev:
  text: Core Concepts
  link: /guide/concepts
next:
  text: Start
  link: /api/start
---

# Go

Spawns a fire-and-forget goroutine with panic recovery, error handling, and telemetry.

## Signature

```go
func Go(ctx context.Context, name string, fn Func, opts ...GoOption)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `name` | `string` | Name for the routine. Used in telemetry spans, metrics, context injection, and logs. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

### Accepted Options

**Shared** (`baseOpt`): `WithTimeout`, `WithRetry`, `WithCircuitBreaker`, `WithFallback`, `WithMiddleware`, `WithLogger`, `WithStallThreshold`, `WithStallHandler`, `WithDurationHistogram`, `WithoutTracing`, `WithoutStartedCounter`, `WithoutErrorCounter`, `WithoutActiveUpDownCounter`, `WithMeterProvider`, `WithTracerProvider`, `WithLimiter`

**Go-only** (`goOnlyOpt`): `WithErrorHandler`, `WithCallerSkip`

See the full [Options reference](/api/options).

## Behavior

1. Options are resolved with defaults (tracing, started counter, error counter, and active counter are enabled).
2. A middleware chain is built: context injection, panic recovery, user middlewares, metrics, tracing, stall detection, timeout.
3. If a `WithLimiter` semaphore is set, it is acquired **before** spawning the goroutine. If the context is cancelled while waiting, the error is handled immediately.
4. The goroutine runs with a derived context (`context.WithCancel`).
5. If `fn` returns an error (or panics), the error is passed to the `ErrorHandler`. The default handler logs via `slog.ErrorContext`.

## Defaults

| Setting | Default |
|---------|---------|
| Tracing | Enabled |
| Started counter | Enabled |
| Error counter | Enabled |
| Active counter | Enabled |
| Duration histogram | Disabled |
| Error handler | `slog.ErrorContext` |

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	gofuncy.Go(ctx, "background-task", func(ctx context.Context) error {
		// Simulate work
		time.Sleep(50 * time.Millisecond)
		fmt.Println("task completed")
		return nil
	},
		gofuncy.WithTimeout(5*time.Second),
	)

	// In production, use proper shutdown signaling instead of sleep
	time.Sleep(100 * time.Millisecond)
}
```

## Types

### Func

```go
type Func func(ctx context.Context) error
```

The function type accepted by `Go`, `Group.Add`, `All`, and `Map`.

### ErrorHandler

```go
type ErrorHandler func(ctx context.Context, err error)
```

Callback for handling errors from fire-and-forget goroutines. Set with `WithErrorHandler`.

### PanicError

```go
type PanicError struct {
    Value any    // the recovered panic value
    Stack []byte // debug.Stack() output
}

func (e *PanicError) Error() string // returns "panic: {value}"
```

Wraps a recovered panic with its stack trace. Use `errors.As` to check for panics:

```go
var pe *gofuncy.PanicError
if errors.As(err, &pe) {
    log.Printf("panic: %v\n%s", pe.Value, pe.Stack)
}
```
