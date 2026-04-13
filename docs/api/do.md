---
prev:
  text: Start
  link: /api/start
next:
  text: Group
  link: /api/group
---

# Do

Executes a function synchronously with the full middleware chain (resilience, telemetry, tracing) and returns the error directly.

## Signature

```go
func Do(ctx context.Context, fn Func, opts ...GoOption) error
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
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.do"` |

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
2. If a `WithLimiter` semaphore is set, it is acquired before execution and released when complete.
3. The function runs **synchronously** on the calling goroutine.
4. The error is returned directly to the caller.

## Do vs Wait vs Go

| | `Do` | `Wait` | `Go` |
|---|---|---|---|
| **Execution** | Synchronous | Async — returns wait function | Async — fire-and-forget |
| **Error handling** | Returns `error` | Wait function returns `error` | `ErrorHandler` callback |
| **Use case** | Inline call with resilience | Launch now, collect result later | Background work |

## Example

```go
// Synchronous call with retry and timeout
err := gofuncy.Do(ctx, func(ctx context.Context) error {
    user, err := api.GetUser(ctx, userID)
    if err != nil {
        return err
    }
    // process user...
    return nil
},
    gofuncy.WithTimeout(5*time.Second),
    gofuncy.WithRetry(3),
)
if err != nil {
    return fmt.Errorf("fetching user: %w", err)
}
```
