---
prev:
  text: Start
  link: /api/start
next:
  text: Start
  link: /api/start
---

# Wait

Spawns a goroutine with the full middleware chain and returns a wait function for deferred result collection.

## Signature

```go
func Wait(ctx context.Context, fn Func, opts ...GoOption) func() error
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

## Options

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.wait"` |

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

### Error Handling

| Option | Description |
|--------|-------------|
| `WithCallerSkip(n)` | Adjust stack depth for span caller attributes. |

## Behavior

1. The full middleware chain is built: panic recovery, resilience (timeout, retry, circuit breaker, fallback), user middlewares, metrics, tracing, stall detection.
2. If a `WithLimiter` semaphore is set, it is acquired before spawning.
3. A goroutine is spawned to execute the function.
4. A wait function is returned immediately.
5. Calling the wait function blocks until the goroutine completes and returns its error.
6. The wait function is safe to call multiple times and from multiple goroutines — it always returns the same result.

## WaitWithStop

Like `Wait`, but the goroutine receives a `StopFunc` it can call to cancel its own context. Returns a wait function for deferred result collection.

### Signature

```go
func WaitWithStop(ctx context.Context, fn func(ctx context.Context, stop StopFunc) error, opts ...GoOption) func() error
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation. |
| `fn` | `func(ctx context.Context, stop StopFunc) error` | The function to execute. Call `stop()` to cancel the goroutine's context. |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

### Behavior

1. The full middleware chain is built, same as `Wait`.
2. A child context with cancel is created before spawning.
3. The cancel function is passed to `fn` as `stop`.
4. A goroutine is spawned to execute the function.
5. A wait function is returned immediately.
6. Calling the wait function blocks until the goroutine completes and returns its error.
7. The wait function is safe to call multiple times and from multiple goroutines.

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.waitwithstop"` |

### Example

```go
wait := gofuncy.WaitWithStop(ctx, func(ctx context.Context, stop StopFunc) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case msg := <-incoming:
            if msg == "done" {
                stop() // self-cancel
            }
            process(msg)
        }
    }
})

// Later, collect the result:
if err := wait(); err != nil {
    log.Println("error:", err)
}
```

## WaitWithReady

Like `Wait`, but the goroutine receives a `ReadyFunc` it can call to signal readiness. The caller blocks until `ready()` is called or the function returns, then receives a wait function for deferred result collection.

### Signature

```go
func WaitWithReady(ctx context.Context, fn func(ctx context.Context, ready ReadyFunc) error, opts ...GoOption) func() error
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation. |
| `fn` | `func(ctx context.Context, ready ReadyFunc) error` | The function to execute. Call `ready()` to signal that initialization is complete. |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

### Behavior

1. The full middleware chain is built, same as `Wait`.
2. A goroutine is spawned to execute the function.
3. The caller blocks until `ready()` is called inside `fn`, or until `fn` returns (whichever comes first).
4. A wait function is returned to the caller.
5. Calling the wait function blocks until the goroutine completes and returns its error.
6. The wait function is safe to call multiple times and from multiple goroutines.

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.waitwithready"` |

### Example

```go
wait := gofuncy.WaitWithReady(ctx, func(ctx context.Context, ready gofuncy.ReadyFunc) error {
    // Perform initialization
    if err := initResources(ctx); err != nil {
        return err
    }
    ready() // signal that initialization is complete

    // Continue running until context is cancelled
    <-ctx.Done()
    return nil
})

// At this point, initialization is guaranteed to be complete (or failed).
// Later, collect the result:
if err := wait(); err != nil {
    log.Println("error:", err)
}
```

## Do vs Wait vs WaitWithStop vs WaitWithReady vs Go

| | `Do` | `Wait` | `WaitWithStop` | `WaitWithReady` | `Go` |
|---|---|---|---|---|---|
| **Execution** | Synchronous | Async — returns wait function | Async — returns wait function | Async — blocks until ready, returns wait function | Async — fire-and-forget |
| **Error handling** | Returns `error` | Wait function returns `error` | Wait function returns `error` | Wait function returns `error` | `ErrorHandler` callback |
| **Stop control** | — | — | Goroutine itself | — | — |
| **Ready signal** | — | — | — | Goroutine signals readiness | — |
| **Use case** | Inline call with resilience | Launch now, collect result later | Self-cancelling goroutine with result | Goroutine with initialization gate | Background work |

## Example

```go
// Launch two async calls
waitUser := gofuncy.Wait(ctx, func(ctx context.Context) error {
    user, err = api.GetUser(ctx, userID)
    return err
}, gofuncy.WithRetry(3))

waitOrders := gofuncy.Wait(ctx, func(ctx context.Context) error {
    orders, err = api.GetOrders(ctx, userID)
    return err
}, gofuncy.WithRetry(3))

// Wait for both
if err := waitUser(); err != nil {
    return err
}
if err := waitOrders(); err != nil {
    return err
}
```
