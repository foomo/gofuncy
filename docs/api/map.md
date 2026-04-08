---
prev:
  text: ForEach
  link: /api/foreach
next:
  text: Options
  link: /api/options
---

# Map

Transforms items concurrently while preserving input order. Uses a `Group` internally.

## Signature

```go
func Map[T, R any](ctx context.Context, name string, items []T, fn func(ctx context.Context, item T) (R, error), opts ...GroupOption) ([]R, error)
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
| `name` | `string` | Name for the operation. Used in telemetry spans, metrics, and logs. |
| `items` | `[]T` | Slice of items to transform. |
| `fn` | `func(ctx context.Context, item T) (R, error)` | Transformation function. |
| `opts` | `...GroupOption` | Options passed to the underlying `NewGroup` call. |

### Return Values

| Value | Description |
|-------|-------------|
| `[]R` | Results in the same order as the input items. On error, elements at failing indices contain the zero value of `R`. |
| `error` | `nil` if all transformations succeed. Otherwise all errors via `errors.Join`. |

If `items` is empty, returns `(nil, nil)` immediately.

### Accepted Options

All `GroupOption` options apply: `WithLimit`, `WithFailFast`, `WithTimeout`, `WithMiddleware`, telemetry options, etc.

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

	users, err := gofuncy.Map(ctx, "fetch-users", ids, func(ctx context.Context, id int) (string, error) {
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
