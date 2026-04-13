---
prev:
  text: Group
  link: /api/group
next:
  text: Map
  link: /api/map
---

# All

Executes a function for each item in a slice concurrently. Uses a `Group` internally.

## Signature

```go
func All[T any](ctx context.Context, items []T, fn func(ctx context.Context, item T) error, opts ...GroupOption) error
```

### Type Parameters

| Parameter | Description |
|-----------|-------------|
| `T` | The type of each item in the slice. |

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. |
| `items` | `[]T` | Slice of items to iterate over. |
| `fn` | `func(ctx context.Context, item T) error` | Function to execute for each item. |
| `opts` | `...GroupOption` | Options passed to the underlying `NewGroup` call. |

### Return Value

Returns `nil` if all invocations succeed. Otherwise returns all errors via `errors.Join`.

If `items` is empty, returns `nil` immediately without creating a group.

## Options

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.all"` |

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

### Group-Only

| Option | Description |
|--------|-------------|
| `WithLimit(n)` | Max concurrent functions in this group. |
| `WithFailFast()` | Cancel remaining functions on first error. |

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

	urls := []string{
		"https://api.example.com/users",
		"https://api.example.com/orders",
		"https://api.example.com/products",
	}

	err := gofuncy.All(ctx, urls, func(ctx context.Context, url string) error {
		fmt.Println("fetching", url)
		// fetch(ctx, url)
		return nil
	},
		gofuncy.WithLimit(2), // at most 2 concurrent fetches
	)
	if err != nil {
		fmt.Println("errors:", err)
	}
}
```

::: tip
`All` is a convenience wrapper around `NewGroup` + `Add` + `Wait`. If you need per-item options or want to add functions dynamically, use `Group` directly.
:::
