---
prev:
  text: Group
  link: /api/group
next:
  text: Map
  link: /api/map
---

# ForEach

Executes a function for each item in a slice concurrently. Uses a `Group` internally.

## Signature

```go
func ForEach[T any](ctx context.Context, name string, items []T, fn func(ctx context.Context, item T) error, opts ...GroupOption) error
```

### Type Parameters

| Parameter | Description |
|-----------|-------------|
| `T` | The type of each item in the slice. |

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Parent context. |
| `name` | `string` | Name for the operation. Used in telemetry spans, metrics, and logs. |
| `items` | `[]T` | Slice of items to iterate over. |
| `fn` | `func(ctx context.Context, item T) error` | Function to execute for each item. |
| `opts` | `...GroupOption` | Options passed to the underlying `NewGroup` call. |

### Return Value

Returns `nil` if all invocations succeed. Otherwise returns all errors via `errors.Join`.

If `items` is empty, returns `nil` immediately without creating a group.

### Accepted Options

All `GroupOption` options apply: `WithLimit`, `WithFailFast`, `WithTimeout`, `WithMiddleware`, telemetry options, etc.

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

	err := gofuncy.ForEach(ctx, "fetch-all", urls, func(ctx context.Context, url string) error {
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
`ForEach` is a convenience wrapper around `NewGroup` + `Add` + `Wait`. If you need per-item options or want to add functions dynamically, use `Group` directly.
:::
