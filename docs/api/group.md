---
prev:
  text: Do
  link: /api/do
next:
  text: All
  link: /api/all
---

# Group

Manages a set of concurrently executing functions with shared lifecycle control, error collection, and optional fail-fast cancellation.

## NewGroup

Creates a new `Group`.

```go
func NewGroup(ctx context.Context, opts ...GroupOption) *Group
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context shared by all functions in the group. |
| `opts` | `...GroupOption` | Functional options. Accepts any `baseOpt` or `groupOnlyOpt`. |

### Options

#### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.FUNC"` |

#### Resilience

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | Per-invocation timeout. Each retry attempt gets a fresh deadline. |
| `WithRetry(n, opts...)` | Automatic retry with configurable backoff. |
| `WithCircuitBreaker(cb)` | Fail fast on broken dependencies. Stateful — share across calls. |
| `WithFallback(fn, opts...)` | Called when the operation fails. Return `nil` to suppress the error. |

#### Telemetry

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

#### Concurrency

| Option | Description |
|--------|-------------|
| `WithLimiter(sem)` | Shared `*semaphore.Weighted` for cross-callsite concurrency control. |

#### Middleware

| Option | Description |
|--------|-------------|
| `WithMiddleware(m...)` | Append custom middleware. Applied after resilience, before telemetry. |
| `WithLogger(l)` | Custom `*slog.Logger` for error reporting. |
| `WithStallThreshold(d)` | Log a warning if the goroutine runs longer than `d`. |
| `WithStallHandler(h)` | Custom callback for stall detection. |

#### Group-Only

| Option | Description |
|--------|-------------|
| `WithLimit(n)` | Max concurrent functions in this group. |
| `WithFailFast()` | Cancel remaining functions on first error. |

## Group.Add

Spawns a goroutine to execute `fn` immediately.

```go
func (g *Group) Add(fn Func, opts ...GoOption)
```

Per-function `opts` are merged on top of the group options. Booleans are OR'd, slices are appended, and non-zero values override. Group-specific fields (`limit`, `failFast`) are not merged.

### Options

#### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.FUNC"` |

#### Resilience

| Option | Description |
|--------|-------------|
| `WithTimeout(d)` | Per-invocation timeout. Each retry attempt gets a fresh deadline. |
| `WithRetry(n, opts...)` | Automatic retry with configurable backoff. |
| `WithCircuitBreaker(cb)` | Fail fast on broken dependencies. Stateful — share across calls. |
| `WithFallback(fn, opts...)` | Called when the operation fails. Return `nil` to suppress the error. |

#### Telemetry

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

#### Concurrency

| Option | Description |
|--------|-------------|
| `WithLimiter(sem)` | Shared `*semaphore.Weighted` for cross-callsite concurrency control. |

#### Middleware

| Option | Description |
|--------|-------------|
| `WithMiddleware(m...)` | Append custom middleware. Applied after resilience, before telemetry. |
| `WithLogger(l)` | Custom `*slog.Logger` for error reporting. |
| `WithStallThreshold(d)` | Log a warning if the goroutine runs longer than `d`. |
| `WithStallHandler(h)` | Custom callback for stall detection. |

#### Error Handling (Go-only)

| Option | Description |
|--------|-------------|
| `WithErrorHandler(h)` | Custom error callback. Default: `slog.ErrorContext`. |
| `WithCallerSkip(n)` | Adjust stack depth for span caller attributes. |

## Group.Wait

Blocks until all added functions complete and returns the joined errors.

```go
func (g *Group) Wait() error
```

Returns `nil` if all functions succeeded, otherwise returns all errors via `errors.Join`.

If tracing is enabled, the group span records all child errors and sets an error status. If `WithDurationHistogram` is set, the group duration is recorded.

If `WithFailFast` was set, the group's context is cancelled after `Wait` returns.

## Behavior

1. `NewGroup` creates the group. If `WithFailFast` is set, a cancellable context is created. If `WithLimit` is set, a buffered channel semaphore is initialized. If tracing is enabled, a span is started.
2. Each `Add` call immediately spawns a goroutine. The function is wrapped with panic recovery, user middlewares, metrics, tracing, stall detection, and timeout (same chain as `Go`).
3. If a `WithLimiter` semaphore is set on the group or per-function, it is acquired before the goroutine starts. If `WithLimit` is set, the internal channel semaphore is used instead.
4. Errors are stored by index. If `WithFailFast` is set, the first error cancels the group context.
5. `Wait` blocks until all goroutines complete, finalizes the span, records the duration histogram, and returns joined errors.

## Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	g := gofuncy.NewGroup(ctx,
		gofuncy.WithLimit(3),       // at most 3 concurrent
		gofuncy.WithFailFast(),     // cancel on first error
	)

	for i := range 10 {
		g.Add(func(ctx context.Context) error {
			fmt.Printf("processing item %d\n", i)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("errors:", err)
	}
}
```

::: tip
Options passed to `Add` are merged on top of group options. This lets you set shared defaults on the group and override per function -- for example, giving each function a unique name while sharing the same timeout.
:::
