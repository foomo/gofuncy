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
func Go(ctx context.Context, fn Func, opts ...GoOption)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

## Options

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.go"` |

### Resilience

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | Per-invocation timeout. Each retry attempt gets a fresh deadline. |
| `WithRetry(n, opts...)` | Automatic retry with configurable backoff. |
| `WithCircuitBreaker(cb)` | Fail fast on broken dependencies. Stateful — share across calls. |
| `WithFallback(fn, opts...)` | Called when the operation fails. Return `nil` to suppress the error. |

### Telemetry

| Option | Default | Description |
|--------|---------|-------------|
| `WithoutTracing()` | on | Disable span creation. |
| `WithDetachedTrace()` | varies | Root span linked to parent instead of child span. Default for `Go`/`Start`/`StartWithReady`/`StartWithStop`/`GoWithCancel`. |
| `WithChildTrace()` | varies | Force child span. Default for `Do`/`Wait`/`NewGroup`. |
| `WithoutStartedCounter()` | on | Disable started counter. |
| `WithoutErrorCounter()` | on | Disable error counter. |
| `WithoutActiveUpDownCounter()` | on | Disable active counter. |
| `WithDurationHistogram()` | off | Enable duration histogram. |
| `WithMeterProvider(mp)` | global | Custom OTel meter provider. |
| `WithTracerProvider(tp)` | global | Custom OTel tracer provider. |

### Concurrency

| Option | Description |
|--------|-------------|
| `WithLimiter(sem)` | Shared `*semaphore.Weighted` for cross-callsite concurrency control. |

### Middleware

| Option | Description |
|--------|-------------|
| `WithMiddleware(m...)` | Append custom middleware. Applied after resilience, before telemetry. |
| `WithLogger(l)` | Custom `*slog.Logger` for error reporting. |
| `WithStallThreshold(d)` | Log a warning if the goroutine runs longer than `d`. |
| `WithStallHandler(h)` | Custom callback for stall detection. |

### Error Handling (Go-only)

| Option | Description |
|--------|-------------|
| `WithErrorHandler(h)` | Custom error callback. Default: `slog.ErrorContext`. |
| `WithCallerSkip(n)` | Adjust stack depth for span caller attributes. |

## Behavior

1. Options are resolved with defaults (tracing, started counter, error counter, and active counter are enabled).
2. A middleware chain is built (innermost to outermost): context injection (applied before the chain), panic recovery, timeout, retry, circuit breaker, fallback, user middlewares, metrics, tracing, stall detection.
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

	gofuncy.Go(ctx, func(ctx context.Context) error {
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

## GoWithCancel

Spawns a goroutine and returns a stop function. Calling stop cancels the goroutine's context, signaling it to shut down. The stop function is safe to call multiple times.

```go
func GoWithCancel(ctx context.Context, fn Func, opts ...GoOption) StopFunc
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. |

### Return Value

Returns a `StopFunc`. Calling `stop()` cancels the goroutine's context, signaling it to shut down. Safe to call multiple times.

### Behavior

1. Same middleware chain as `Go`.
2. A child context with cancel is created before spawning.
3. The returned function cancels this context.
4. Errors are handled via the `ErrorHandler`.

### Example

```go
stop := gofuncy.GoWithCancel(ctx, func(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil // clean shutdown
        case <-time.After(time.Second):
            process()
        }
    }
})

// Later, when you want to stop:
stop()
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
