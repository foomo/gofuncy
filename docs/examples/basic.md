---
prev:
  text: Channel
  link: /api/channel

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

## Async with Deferred Result

Launch work now, collect the result when you need it. The wait function is safe to call from multiple goroutines.

```go
package main

import (
	"context"
	"fmt"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	var user string
	var orders []string

	// Launch two async calls
	waitUser := gofuncy.Wait(ctx, "fetch-user", func(ctx context.Context) error {
		user = "Alice"
		return nil
	})

	waitOrders := gofuncy.Wait(ctx, "fetch-orders", func(ctx context.Context) error {
		orders = []string{"order-1", "order-2"}
		return nil
	})

	// Wait for both
	if err := waitUser(); err != nil {
		fmt.Println("user error:", err)
		return
	}
	if err := waitOrders(); err != nil {
		fmt.Println("orders error:", err)
		return
	}

	fmt.Printf("%s has %d orders\n", user, len(orders))
	// Output: Alice has 2 orders
}
```

## Synchronous Execution with Do

Run a function through the full middleware chain without spawning a goroutine. Useful for inline calls that need retry, timeout, or circuit breaker.

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

	err := gofuncy.Do(ctx, "fetch-config", func(ctx context.Context) error {
		// Simulate a flaky call
		return fmt.Errorf("connection refused")
	},
		gofuncy.WithRetry(3, gofuncy.RetryBackoff(gofuncy.BackoffConstant(100*time.Millisecond))),
		gofuncy.WithTimeout(500*time.Millisecond),
	)
	if err != nil {
		fmt.Println("failed after retries:", err)
	}
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

## All Over a Slice

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

	err := gofuncy.All(ctx, "process-items", items, func(ctx context.Context, item string) error {
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

## Channel

Send and receive values through an observable channel with built-in metrics.

```go
package main

import (
	"context"
	"fmt"

	"github.com/foomo/gofuncy/channel"
)

func main() {
	ctx := context.Background()

	ch := channel.New[int]("numbers", channel.WithBuffer[int](5))

	// Send multiple values at once
	if err := ch.Send(ctx, 1, 2, 3, 4, 5); err != nil {
		fmt.Println("send error:", err)
		return
	}

	fmt.Printf("buffered: %d/%d\n", ch.Len(), ch.Cap())
	// Output: buffered: 5/5

	// Close and drain
	ch.Close()

	for v := range ch.Receive() {
		fmt.Println(v)
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
