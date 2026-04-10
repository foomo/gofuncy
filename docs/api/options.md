---
prev:
  text: Map
  link: /api/map
next:
  text: Channel
  link: /api/channel

---

# Options

All options are configured via functional option constructors. Options fall into three categories based on which functions accept them.

## Option Categories

| Category | Interface | Accepted by |
|----------|-----------|-------------|
| Shared | `GoOption` + `GroupOption` | `Do`, `Wait`, `Go`, `NewGroup`, `Group.Add` |
| Go-only | `GoOption` | `Do`, `Wait`, `Go`, `Group.Add` |
| Group-only | `GroupOption` | `NewGroup`, `All`, `Map` |

The compiler enforces these constraints. You cannot pass a `groupOnlyOpt` to `Go()` or a `goOnlyOpt` to `NewGroup()`.

## Resilience Options

These options configure the built-in resilience chain. The framework applies them in the correct order automatically:

```
fn → timeout → retry → circuitBreaker → fallback
```

### WithTimeout

```go
func WithTimeout(timeout time.Duration) baseOpt
```

Sets a per-invocation timeout. When combined with `WithRetry`, each retry attempt gets its own fresh deadline.

```go
g.Add("fetch", fetchData,
    gofuncy.WithTimeout(5*time.Second),
    gofuncy.WithRetry(3),
)
// Each of the 3 attempts gets 5s
```

### WithRetry

```go
func WithRetry(maxAttempts int, opts ...RetryOption) baseOpt
```

Configures automatic retry. `maxAttempts` is the total number of attempts (1 = no retry, 3 = initial + 2 retries).

```go
g.Add("fetch", fetchData,
    gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(time.Second))),
)
```

### WithCircuitBreaker

```go
func WithCircuitBreaker(cb *CircuitBreaker) baseOpt
```

Sets a circuit breaker for the operation. The circuit breaker is stateful — create one via `NewCircuitBreaker` and share it across all calls to the same dependency.

```go
var apiBreaker = gofuncy.NewCircuitBreaker(
    gofuncy.CircuitBreakerThreshold(5),
    gofuncy.CircuitBreakerCooldown(30*time.Second),
)

g.Add("api-call", callAPI, gofuncy.WithCircuitBreaker(apiBreaker))
```

### WithFallback

```go
func WithFallback(fn func(ctx context.Context, err error) error, opts ...FallbackOption) baseOpt
```

Sets a fallback function that is called when the operation fails. The fallback receives the original error and may return `nil` to suppress it or a different error.

```go
g.Add("fetch", fetchData,
    gofuncy.WithFallback(func(ctx context.Context, err error) error {
        return loadFromCache(ctx)
    }),
)
```

### Resilience Chain Order

The framework applies resilience options in a fixed order, regardless of the order they appear in your code:

| Position | Middleware | Behavior |
|----------|-----------|----------|
| Innermost | **Timeout** | Each invocation gets a fresh deadline |
| ↑ | **Retry** | Retries the timeout-wrapped function |
| ↑ | **Circuit Breaker** | Sees the final outcome after all retries |
| Outermost | **Fallback** | Last resort — catches everything |

For custom ordering, use `WithMiddleware` with the middleware constructors (`Retry()`, `Fallback()`, etc.) directly.

## Shared Options

These options implement both `GoOption` and `GroupOption`.

### WithLogger

```go
func WithLogger(l *slog.Logger) baseOpt
```

Configures the logger for error reporting and stall detection warnings.

### WithMiddleware

```go
func WithMiddleware(m ...Middleware) baseOpt
```

Appends middleware to the operation's middleware chain. User middlewares are applied **after** the built-in resilience chain (timeout, retry, circuit breaker, fallback) and **before** telemetry.

```go
type Middleware func(Func) Func
```

### WithStallThreshold

```go
func WithStallThreshold(d time.Duration) baseOpt
```

Enables stall detection. If a goroutine runs longer than the threshold:
- A warning is logged via `slog` (or the custom `StallHandler`)
- The `gofuncy.goroutines.stalled` metric is incremented
- The goroutine is **not** cancelled

### WithStallHandler

```go
func WithStallHandler(h StallHandler) baseOpt
```

Sets a custom callback for stall detection. If not set, stalls are logged via `slog`.

```go
type StallHandler func(ctx context.Context, name string, threshold time.Duration)
```

### WithLimiter

```go
func WithLimiter(l *semaphore.Weighted) baseOpt
```

Sets a shared weighted semaphore for concurrency control. The semaphore is acquired before the goroutine starts and released when it completes. Use `semaphore.NewWeighted(n)` from `golang.org/x/sync/semaphore` to create one.

Unlike `WithLimit` (which is per-group), `WithLimiter` can be shared across multiple `Go` calls and groups.

### WithDurationHistogram

```go
func WithDurationHistogram() baseOpt
```

Enables the `gofuncy.goroutines.duration.seconds` histogram metric for individual goroutine execution time. For groups, also enables `gofuncy.groups.duration.seconds`.

Disabled by default to reduce metric cardinality.

### WithoutTracing

```go
func WithoutTracing() baseOpt
```

Disables OpenTelemetry span creation for the operation. Tracing is enabled by default.

### WithDetachedTrace

```go
func WithDetachedTrace() baseOpt
```

Creates new root spans linked to the parent span context instead of child spans. This is useful when goroutines represent independent work units (e.g., event processing) that should not be nested under the caller's trace but should still reference it.

For `Go()`, detached traces are the default — use `WithChildTrace` to opt out. For `Do()`, `Wait()`, and `NewGroup()`, child traces are the default.

### WithChildTrace

```go
func WithChildTrace() baseOpt
```

Forces child spans instead of detached root spans. This is primarily useful with `Go()` to override its default detached behavior.

### WithoutStartedCounter

```go
func WithoutStartedCounter() baseOpt
```

Disables the `gofuncy.goroutines.started` counter metric. Enabled by default.

### WithoutErrorCounter

```go
func WithoutErrorCounter() baseOpt
```

Disables the `gofuncy.goroutines.errors` counter metric. Enabled by default.

### WithoutActiveUpDownCounter

```go
func WithoutActiveUpDownCounter() baseOpt
```

Disables the `gofuncy.goroutines.active` up-down counter metric. Enabled by default.

### WithMeterProvider

```go
func WithMeterProvider(mp metric.MeterProvider) baseOpt
```

Sets a custom OpenTelemetry meter provider. Defaults to `otel.GetMeterProvider()`.

### WithTracerProvider

```go
func WithTracerProvider(tp trace.TracerProvider) baseOpt
```

Sets a custom OpenTelemetry tracer provider. Defaults to `otel.GetTracerProvider()`.

## Go-Only Options

These options implement only `GoOption` and are accepted by `Do`, `Wait`, `Go`, and `Group.Add`.

### WithErrorHandler

```go
func WithErrorHandler(h ErrorHandler) goOnlyOpt
```

Sets a custom error handler callback. If not set, errors are logged via `slog.ErrorContext`.

```go
type ErrorHandler func(ctx context.Context, err error)
```

### WithCallerSkip

```go
func WithCallerSkip(skip int) goOnlyOpt
```

Sets the caller skip for error reporting in traces. Adjusts the stack depth for finding the actual call site in span attributes.

## Group-Only Options

These options implement only `GroupOption` and are accepted by `NewGroup` (and therefore `All` and `Map`).

### WithLimit

```go
func WithLimit(n int) groupOnlyOpt
```

Sets the maximum number of concurrently executing functions in a group. Uses an internal buffered channel as a semaphore. Unlike `WithLimiter`, this semaphore is scoped to the group.

### WithFailFast

```go
func WithFailFast() groupOnlyOpt
```

Configures the group to cancel remaining functions on first error. Creates a cancellable context that is cancelled when any added function returns a non-nil error. Remaining functions will see `ctx.Err() != nil`.

## Defaults

When no options are passed, the following defaults apply:

| Setting | Default |
|---------|---------|
| Tracing | Enabled |
| Started counter | Enabled |
| Error counter | Enabled |
| Active counter | Enabled |
| Duration histogram | Disabled |
| Error handler (`Go` only) | `slog.ErrorContext` |
| Limit | No limit |
| Fail-fast | Disabled |
| Retry | Disabled |
| Circuit breaker | Disabled |
| Fallback | Disabled |

## Option Merging in Group.Add

When you pass options to `Group.Add`, they are merged on top of the group's options using these rules:

| Field type | Merge rule |
|-----------|------------|
| `*slog.Logger` | Override if non-nil |
| `[]Middleware` | Append |
| `time.Duration` | Override if > 0 |
| `bool` (metrics/tracing) | OR (enable, never disable) |
| `MeterProvider` / `TracerProvider` | Override if non-nil |
| `*semaphore.Weighted` | Override if non-nil |
| `*CircuitBreaker` | Override if non-nil |
| Retry / Fallback | Override if set |
| `limit`, `failFast` | Not merged (group-only) |

Note: The `name` parameter passed to `Add` always takes precedence over the group name.
