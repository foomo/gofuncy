---
prev:
  text: Go
  link: /api/go
next:
  text: Do
  link: /api/do
---

# Wait

Spawns a goroutine with the full middleware chain and returns a wait function for deferred result collection.

## Signature

```go
func Wait(ctx context.Context, name string, fn Func, opts ...GoOption) func() error
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation. |
| `name` | `string` | Name for the routine. Used in telemetry spans, metrics, context injection, and logs. |
| `fn` | `Func` | The function to execute. Signature: `func(ctx context.Context) error` |
| `opts` | `...GoOption` | Functional options. Accepts any `baseOpt` or `goOnlyOpt`. |

### Accepted Options

**Shared** (`baseOpt`): `WithTimeout`, `WithRetry`, `WithCircuitBreaker`, `WithFallback`, `WithMiddleware`, `WithLogger`, `WithStallThreshold`, `WithStallHandler`, `WithDurationHistogram`, `WithoutTracing`, `WithoutStartedCounter`, `WithoutErrorCounter`, `WithoutActiveUpDownCounter`, `WithMeterProvider`, `WithTracerProvider`, `WithLimiter`

**Go-only** (`goOnlyOpt`): `WithCallerSkip`

See the full [Options reference](/api/options).

## Behavior

1. The full middleware chain is built: panic recovery, resilience (timeout, retry, circuit breaker, fallback), user middlewares, metrics, tracing, stall detection.
2. If a `WithLimiter` semaphore is set, it is acquired before spawning.
3. A goroutine is spawned to execute the function.
4. A wait function is returned immediately.
5. Calling the wait function blocks until the goroutine completes and returns its error.
6. The wait function is safe to call multiple times and from multiple goroutines — it always returns the same result.

## Do vs Wait vs Go

| | `Do` | `Wait` | `Go` |
|---|---|---|---|
| **Execution** | Synchronous | Async — returns wait function | Async — fire-and-forget |
| **Error handling** | Returns `error` | Wait function returns `error` | `ErrorHandler` callback |
| **Use case** | Inline call with resilience | Launch now, collect result later | Background work |

## Example

```go
// Launch two async calls
waitUser := gofuncy.Wait(ctx, "fetch-user", func(ctx context.Context) error {
    user, err = api.GetUser(ctx, userID)
    return err
}, gofuncy.WithRetry(3))

waitOrders := gofuncy.Wait(ctx, "fetch-orders", func(ctx context.Context) error {
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
