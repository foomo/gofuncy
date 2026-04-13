---
prev:
  text: Go
  link: /api/go
next:
  text: Wait
  link: /api/wait
---

# Start / StartWithReady / StartWithStop

Spawn goroutines with startup guarantees. `Start` blocks until the goroutine is scheduled. `StartWithReady` blocks until the goroutine signals readiness. `StartWithStop` passes a stop function into the goroutine.

## Start

Blocks until the goroutine has actually started executing. Useful in tests where you need the routine to be running before proceeding.

```go
func Start(ctx context.Context, fn Func, opts ...GoOption)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. |

### Behavior

1. Same middleware chain as [`Go`](/api/go): panic recovery, resilience, user middlewares, metrics, tracing, stall detection.
2. If a `WithLimiter` semaphore is set, it is acquired before spawning.
3. The goroutine is spawned and `Start` blocks until it begins executing.
4. Errors are handled via the `ErrorHandler` (default: `slog.ErrorContext`).

### Example

```go
gofuncy.Start(ctx, func(ctx context.Context) error {
    // This goroutine is guaranteed to be running
    // by the time Start returns.
    return serve(ctx)
})
// Safe to send requests — the server is running.
```

## StartWithReady

Blocks until the goroutine signals readiness by calling `ready()`. If the function returns before calling `ready()`, `StartWithReady` unblocks anyway.

```go
func StartWithReady(ctx context.Context, fn func(ctx context.Context, ready ReadyFunc) error, opts ...GoOption)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `fn` | `func(ctx context.Context, ready ReadyFunc) error` | The function to execute. Call `ready()` to signal readiness. |
| `opts` | `...GoOption` | Functional options. |

### Behavior

1. Same middleware chain as `Go` and `Start`.
2. `StartWithReady` blocks until either `ready()` is called or `fn` returns — whichever comes first.
3. The `ready` function is safe to call multiple times (protected by `sync.Once`).
4. If `fn` panics or returns an error before calling `ready()`, `StartWithReady` still unblocks and the error is routed to the error handler.

### Example

```go
gofuncy.StartWithReady(ctx, func(ctx context.Context, ready ReadyFunc) error {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        return err
    }
    ready() // signal: listener is up
    return http.Serve(ln, handler)
})
// At this point, the listener is guaranteed to be accepting connections.
```

## StartWithStop

Spawns a fire-and-forget goroutine that receives a stop function. Calling stop cancels the goroutine's context from within, signaling it to shut down. The stop function is safe to call multiple times.

```go
func StartWithStop(ctx context.Context, fn func(ctx context.Context, stop StopFunc) error, opts ...GoOption)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. Cancellation is propagated to the goroutine. |
| `fn` | `func(ctx context.Context, stop StopFunc) error` | The function to execute. Call `stop()` to cancel the goroutine's context. |
| `opts` | `...GoOption` | Functional options. |

### Behavior

1. Same middleware chain as `Go`.
2. A child context with cancel is created before spawning.
3. The cancel function is passed to `fn` as `stop`.
4. Fire-and-forget — does not block the caller.
5. Errors are handled via the `ErrorHandler`.

### Example

```go
gofuncy.StartWithStop(ctx, func(ctx context.Context, stop StopFunc) error {
    for {
        select {
        case <-ctx.Done():
            return nil // clean shutdown
        case msg := <-incoming:
            if msg == "quit" {
                stop() // self-cancel
            }
        }
    }
})
```

## Comparison

| | `Go` | `Start` | `StartWithReady` | `StartWithStop` |
|---|---|---|---|---|
| **Blocks until** | Never | Goroutine is scheduled | `ready()` is called | Never |
| **Returns** | Nothing | Nothing | Nothing | Nothing |
| **Stop control** | — | — | — | Goroutine itself |
| **Use case** | Background work | Tests needing startup guarantee | Initialization-dependent work | Self-cancelling goroutine |
| **Error handling** | `ErrorHandler` | `ErrorHandler` | `ErrorHandler` | `ErrorHandler` |

## Options

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.start"` / `"gofuncy.startwithready"` / `"gofuncy.startwithstop"` |

### Resilience

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | Per-invocation timeout. Each retry attempt gets a fresh deadline. |
| `WithRetry(n, opts...)` | Automatic retry with configurable backoff. |
| `WithCircuitBreaker(cb)` | Fail fast on broken dependencies. Stateful -- share across calls. |
| `WithFallback(fn, opts...)` | Called when the operation fails. Return `nil` to suppress the error. |

### Telemetry

| Option | Default | Description |
|--------|---------|-------------|
| `WithoutTracing()` | on | Disable span creation. |
| `WithDetachedTrace()` | **on** | Root span linked to parent instead of child span. Default for `Start`/`StartWithReady`/`StartWithStop`. |
| `WithChildTrace()` | off | Force child span. |
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
