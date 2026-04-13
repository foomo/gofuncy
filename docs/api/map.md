---
prev:
  text: All
  link: /api/all
next:
  text: Options
  link: /api/options

---

# Map

Transforms items concurrently while preserving input order. Uses a `Group` internally.

## Signature

```go
func Map[T, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error), opts ...GroupOption) ([]R, error)
```

### Type Parameters

| Parameter | Description |
|-----------|-------------|
| `T` | Input item type. |
| `R` | Output result type. |

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. |
| `items` | `[]T` | Slice of items to transform. |
| `fn` | `func(ctx context.Context, item T) (R, error)` | Transformation function. |
| `opts` | `...GroupOption` | Options passed to the underlying `NewGroup` call. |

### Return Values

| Value | Description |
|-------|-------------|
| `[]R` | Results in the same order as the input items. On error, elements at failing indices contain the zero value of `R`. |
| `error` | `nil` if all transformations succeed. Otherwise all errors via `errors.Join`. |

If `items` is empty, returns `(nil, nil)` immediately.

## Options

### Naming

| Option | Description |
|--------|-------------|
| `WithName(name)` | Custom metric/tracing label. Default: `"gofuncy.map"` |

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
	"strings"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	ids := []int{1, 2, 3, 4, 5}

	users, err := gofuncy.Map(ctx, ids, func(ctx context.Context, id int) (string, error) {
		// Simulate fetching a user by ID
		return fmt.Sprintf("user-%d", id), nil
	},
		gofuncy.WithLimit(3),
	)
	if err != nil {
		fmt.Println("errors:", err)
	}

	fmt.Println(strings.Join(users, ", "))
	// Output: user-1, user-2, user-3, user-4, user-5
}
```

::: warning
Results are stored by index. If some transformations fail and you did not use `WithFailFast`, the result slice will contain zero values at the indices of failed items. Always check the returned error.
:::

::: tip
`Map` preserves order by storing results at the corresponding index. There is no need to sort or reorder results after the call.
:::
