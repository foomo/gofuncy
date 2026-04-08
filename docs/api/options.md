---
prev:
  text: Map
  link: /api/map
next:
  text: Basic Examples
  link: /examples/basic
---

# Options

All options are configured via functional option constructors. Options fall into three categories based on which functions accept them.

## Option Categories

| Category | Interface | Accepted by |
|----------|-----------|-------------|
| Shared | `GoOption` + `GroupOption` | `Go`, `NewGroup`, `Group.Add` |
| Go-only | `GoOption` | `Go`, `Group.Add` |
| Group-only | `GroupOption` | `NewGroup`, `ForEach`, `Map` |

The compiler enforces these constraints. You cannot pass a `groupOnlyOpt` to `Go()` or a `goOnlyOpt` to `NewGroup()`.

## Shared Options

These options implement both `GoOption` and `GroupOption`.

### WithLogger

```go
func WithLogger(l *slog.Logger) baseOpt
```

Configures the logger for error reporting and stall detection warnings.

### WithTimeout

```go
func WithTimeout(timeout time.Duration) baseOpt
```

Sets a timeout for the operation. Applies `context.WithTimeout` to the execution context. The goroutine's context is cancelled when the timeout elapses.

### WithMiddleware

```go
func WithMiddleware(m ...Middleware) baseOpt
```

Appends middleware to the operation's middleware chain. Middlewares wrap the function and execute from outermost to innermost.

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
type StallHandler func(ctx context.Context, name string, elapsed time.Duration)
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

These options implement only `GoOption` and are accepted by `Go` and `Group.Add`.

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

These options implement only `GroupOption` and are accepted by `NewGroup` (and therefore `ForEach` and `Map`).

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
| `limit`, `failFast` | Not merged (group-only) |

Note: The `name` parameter passed to `Add` always takes precedence over the group name.
