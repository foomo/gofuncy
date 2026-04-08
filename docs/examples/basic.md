---
prev:
  text: Options
  link: /api/options
next:
  text: Advanced Examples
  link: /examples/advanced
---

# Basic Examples

## Fire-and-Forget

Spawn a background goroutine. Errors are logged automatically.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	gofuncy.Go(ctx, "fire-and-forget", func(ctx context.Context) error {
		fmt.Println("running in the background")
		return nil
	})

	time.Sleep(100 * time.Millisecond)
}
```

## Custom Error Handler

Override the default slog handler to handle errors yourself.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	gofuncy.Go(ctx, "custom-handler", func(ctx context.Context) error {
		return errors.New("something went wrong")
	},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			fmt.Printf("[%s] error: %v\n", gofuncy.NameFromContext(ctx), err)
		}),
	)

	time.Sleep(100 * time.Millisecond)
	// Output: [custom-handler] error: something went wrong
}
```

## Basic Group

Run multiple functions concurrently and collect all errors.

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	g := gofuncy.NewGroup(ctx, "parallel-tasks")

	g.Add("task-a", func(ctx context.Context) error {
		fmt.Println("task A")
		return nil
	})

	g.Add("task-b", func(ctx context.Context) error {
		fmt.Println("task B")
		return errors.New("task B failed")
	})

	g.Add("task-c", func(ctx context.Context) error {
		fmt.Println("task C")
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Println("group error:", err)
	}
	// Output (order may vary):
	// task A
	// task B
	// task C
	// group error: task B failed
}
```

## ForEach Over a Slice

Iterate over items concurrently with a concurrency limit.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	items := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	err := gofuncy.ForEach(ctx, "process-items", items, func(ctx context.Context, item string) error {
		fmt.Printf("processing %s\n", item)
		time.Sleep(50 * time.Millisecond) // simulate work
		return nil
	},
		gofuncy.WithLimit(2), // process 2 at a time
	)
	if err != nil {
		fmt.Println("errors:", err)
	}
}
```

## Map with Result Collection

Transform items concurrently and collect results in order.

```go
package main

import (
	"context"
	"fmt"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	numbers := []int{1, 2, 3, 4, 5}

	squares, err := gofuncy.Map(ctx, "square", numbers, func(ctx context.Context, n int) (int, error) {
		return n * n, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(squares) // [1 4 9 16 25]
}
```
