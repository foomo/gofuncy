---
prev:
  text: Basic Examples
  link: /examples/basic
next:
  text: Patterns
  link: /examples/patterns
---

# Advanced Examples

## Group with Semaphore Limit and Fail-Fast

Limit concurrent execution and stop early on the first error.

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

	g := gofuncy.NewGroup(ctx, "bounded-group",
		gofuncy.WithLimit(5),
		gofuncy.WithFailFast(),
	)

	for i := range 20 {
		g.Add(fmt.Sprintf("task-%d", i), func(ctx context.Context) error {
			// Check context before doing work -- fail-fast cancels the context
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if i == 7 {
				return errors.New("task 7 failed")
			}

			fmt.Printf("completed task %d\n", i)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("group error:", err)
	}
}
```

## Shared Limiter Across Call Sites

Use a `*semaphore.Weighted` to limit concurrency across multiple independent `Go` calls.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
	"golang.org/x/sync/semaphore"
)

func main() {
	ctx := context.Background()

	// At most 3 goroutines running at any time, across all call sites
	limiter := semaphore.NewWeighted(3)

	for i := range 10 {
		gofuncy.Go(ctx, fmt.Sprintf("worker-%d", i), func(ctx context.Context) error {
			fmt.Printf("worker %d started\n", i)
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("worker %d done\n", i)
			return nil
		},
			gofuncy.WithLimiter(limiter),
		)
	}

	time.Sleep(500 * time.Millisecond)
}
```

## Stall Detection

Detect goroutines that take longer than expected without cancelling them.

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

	gofuncy.Go(ctx, "slow-task", func(ctx context.Context) error {
		// This takes longer than the stall threshold
		time.Sleep(3 * time.Second)
		return nil
	},
		gofuncy.WithStallThreshold(1*time.Second),
		gofuncy.WithStallHandler(func(ctx context.Context, name string, elapsed time.Duration) {
			fmt.Printf("STALL: %s has been running for %v\n", name, elapsed)
		}),
	)

	time.Sleep(4 * time.Second)
	// Output: STALL: slow-task has been running for 1s
}
```

## Custom Middleware

Add cross-cutting behavior like logging or retries using middleware.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

// loggingMiddleware logs the start and end of each function execution.
func loggingMiddleware(next gofuncy.Func) gofuncy.Func {
	return func(ctx context.Context) error {
		name := gofuncy.NameFromContext(ctx)
		fmt.Printf("[%s] starting\n", name)
		start := time.Now()

		err := next(ctx)

		dur := time.Since(start)
		if err != nil {
			fmt.Printf("[%s] failed after %v: %v\n", name, dur, err)
		} else {
			fmt.Printf("[%s] completed in %v\n", name, dur)
		}
		return err
	}
}

func main() {
	ctx := context.Background()

	g := gofuncy.NewGroup(ctx, "with-logging",
		gofuncy.WithMiddleware(loggingMiddleware),
	)

	g.Add("task-a", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	g.Add("task-b", func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	_ = g.Wait()
	// Output:
	// [task-a] starting
	// [task-b] starting
	// [task-a] completed in 50ms
	// [task-b] completed in 100ms
}
```

## Map with Timeout

Transform items concurrently with a per-group timeout.

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

	ids := []int{1, 2, 3, 4, 5}

	results, err := gofuncy.Map(ctx, "fetch-with-timeout", ids, func(ctx context.Context, id int) (string, error) {
		// Simulate variable latency
		time.Sleep(time.Duration(id*100) * time.Millisecond)

		// Respect context cancellation
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		return fmt.Sprintf("result-%d", id), nil
	},
		gofuncy.WithTimeout(250*time.Millisecond),
		gofuncy.WithLimit(3),
	)

	fmt.Println("results:", results)
	if err != nil {
		fmt.Println("errors:", err)
	}
}
```

## Disabling Telemetry

Turn off all OpenTelemetry instrumentation for performance-critical paths.

```go
package main

import (
	"context"

	"github.com/foomo/gofuncy"
)

func main() {
	ctx := context.Background()

	gofuncy.Go(ctx, "hot-path", func(ctx context.Context) error {
		// Hot path -- no telemetry overhead
		return nil
	},
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	)
}
```
