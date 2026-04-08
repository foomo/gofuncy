---
prev:
  text: Go
  link: /api/go
next:
  text: ForEach
  link: /api/foreach
---

# Group

Manages a set of concurrently executing functions with shared lifecycle control, error collection, and optional fail-fast cancellation.

## NewGroup

Creates a new `Group`.

```go
func NewGroup(ctx context.Context, name string, opts ...GroupOption) *Group
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context shared by all functions in the group. |
| `name` | `string` | Name for the group. Used in telemetry spans, metrics, and logs. |
| `opts` | `...GroupOption` | Functional options. Accepts any `baseOpt` or `groupOnlyOpt`. |

### Accepted Options

**Shared** (`baseOpt`): `WithLogger`, `WithTimeout`, `WithMiddleware`, `WithStallThreshold`, `WithStallHandler`, `WithDurationHistogram`, `WithoutTracing`, `WithoutStartedCounter`, `WithoutErrorCounter`, `WithoutActiveUpDownCounter`, `WithMeterProvider`, `WithTracerProvider`, `WithLimiter`

**Group-only** (`groupOnlyOpt`): `WithLimit`, `WithFailFast`

## Group.Add

Spawns a goroutine to execute `fn` immediately.

```go
func (g *Group) Add(name string, fn Func, opts ...GoOption)
```

Per-function `opts` are merged on top of the group options. Booleans are OR'd, slices are appended, and non-zero values override. Group-specific fields (`limit`, `failFast`) are not merged.

The `name` parameter sets the name for this specific function, overriding the group name for telemetry and logging purposes.

### Accepted Options

**Shared** (`baseOpt`): all shared options

**Go-only** (`goOnlyOpt`): `WithErrorHandler`, `WithCallerSkip`

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

	g := gofuncy.NewGroup(ctx, "data-pipeline",
		gofuncy.WithLimit(3),       // at most 3 concurrent
		gofuncy.WithFailFast(),     // cancel on first error
	)

	for i := range 10 {
		g.Add(fmt.Sprintf("item-%d", i), func(ctx context.Context) error {
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
