---
prev:
  text: Wait
  link: /api/wait
next:
  text: Group
  link: /api/group
---

# Do

Executes a function synchronously with the full middleware chain (resilience, telemetry, tracing) and returns the error directly.

## Signature

```go
func Do(ctx context.Context, name string, fn Func, opts ...GoOption) error
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation. |
| `name` | `string` | Name for the routine. Used in telemetry spans, metrics, context injection, and logs. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

### Accepted Options

**Shared** (`baseOpt`): `WithTimeout`, `WithRetry`, `WithCircuitBreaker`, `WithFallback`, `WithMiddleware`, `WithLogger`, `WithStallThreshold`, `WithStallHandler`, `WithDurationHistogram`, `WithoutTracing`, `WithDetachedTrace`, `WithChildTrace`, `WithoutStartedCounter`, `WithoutErrorCounter`, `WithoutActiveUpDownCounter`, `WithMeterProvider`, `WithTracerProvider`, `WithLimiter`

**Go-only** (`goOnlyOpt`): `WithCallerSkip`

See the full [Options reference](/api/options).

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
err := gofuncy.Do(ctx, "fetch-user", func(ctx context.Context) error {
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
