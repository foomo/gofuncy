---
prev:
  text: Getting Started
  link: /guide/getting-started
next:
  text: Go API
  link: /api/go
---

# Core Concepts

## The Options Pattern

gofuncy uses functional options to configure behavior. There are three option categories:

| Type | Applies to | Example |
|------|-----------|---------|
| `baseOpt` | `Go`, `NewGroup`, `Group.Add` | `WithTimeout`, `WithMiddleware`, `WithLogger` |
| `goOnlyOpt` | `Go`, `Group.Add` | `WithErrorHandler`, `WithCallerSkip` |
| `groupOnlyOpt` | `NewGroup` | `WithLimit`, `WithFailFast` |

Options are implemented as interfaces (`GoOption` and `GroupOption`), so the compiler prevents you from passing a group-only option to `Go()` or a go-only option to `NewGroup()`.

The `name` parameter is required on all functions and is not an option:

```go
// Name is always the second argument (after context)
gofuncy.Go(ctx, "worker", fn, gofuncy.WithTimeout(5*time.Second))

// Group-only options work with NewGroup
g := gofuncy.NewGroup(ctx, "pipeline", gofuncy.WithLimit(10), gofuncy.WithFailFast())

// Per-function name override with Group.Add
g.Add("specific-task", fn)
```

When you pass options to `Group.Add`, they are **merged** on top of the group's options. Booleans are OR'd, slices are appended, and non-zero values override.

See the full [Options reference](/api/options) for every available option.

## Error Handling

### Panic Recovery

Every goroutine spawned by gofuncy automatically recovers from panics. A recovered panic is wrapped in a `*PanicError` that preserves the original value and the full stack trace:

```go
type PanicError struct {
    Value any    // the recovered panic value
    Stack []byte // runtime/debug.Stack() output
}
```

You can check for panics using `errors.As`:

```go
var panicErr *gofuncy.PanicError
if errors.As(err, &panicErr) {
    fmt.Printf("panic: %v\nstack:\n%s\n", panicErr.Value, panicErr.Stack)
}
```

### Error Collection

- `Go` -- errors are passed to an `ErrorHandler` callback. The default handler logs via `slog.ErrorContext`. Override with `WithErrorHandler`.
- `Group.Wait` -- returns all errors from added functions via `errors.Join`.
- `ForEach` and `Map` -- return all errors via `errors.Join`.

### Fail-Fast

Pass `WithFailFast()` to `NewGroup` to cancel all remaining functions when the first error occurs. The group's context is cancelled, so functions that check `ctx.Err()` will exit early.

## Context and Naming

gofuncy injects routine metadata into the context automatically:

```go
// Extract the routine name (defaults to "noname")
name := gofuncy.NameFromContext(ctx)

// Extract the parent routine name (empty if none)
parent := gofuncy.ParentFromContext(ctx)
```

You can also use the `Context` helper:

```go
c := gofuncy.Ctx(ctx)
c.Name()   // routine name
c.Parent() // parent name
c.Root()   // returns context with name set to "root"
```

Names are used in OpenTelemetry spans and metrics attributes, making it easy to trace goroutine hierarchies in your observability stack.

## Concurrency Control

gofuncy offers two levels of concurrency limiting:

### Per-Group: WithLimit

Limits the number of concurrently executing functions within a single group. Uses an internal buffered channel as a semaphore.

```go
// At most 5 functions run at the same time
g := gofuncy.NewGroup(ctx, "workers", gofuncy.WithLimit(5))
```

### Cross-Callsite: WithLimiter

Shares a `*semaphore.Weighted` across multiple `Go` or `Group.Add` calls, even across different groups. The semaphore is acquired before the goroutine starts and released when it completes.

```go
import "golang.org/x/sync/semaphore"

// Global limiter: at most 20 concurrent goroutines across all call sites
limiter := semaphore.NewWeighted(20)

gofuncy.Go(ctx, "task-1", fn1, gofuncy.WithLimiter(limiter))
gofuncy.Go(ctx, "task-2", fn2, gofuncy.WithLimiter(limiter))
```

::: warning
`WithLimiter` acquires the semaphore **before** spawning the goroutine. If the context is cancelled while waiting, the error is handled immediately and the goroutine is not started.
:::

## Middleware

The `Middleware` type wraps a `Func` to add cross-cutting behavior:

```go
type Middleware func(Func) Func
```

Middlewares are applied in order via `WithMiddleware`. They compose into a chain that executes from outermost to innermost:

```go
logging := func(next gofuncy.Func) gofuncy.Func {
    return func(ctx context.Context) error {
        fmt.Println("start")
        err := next(ctx)
        fmt.Println("done")
        return err
    }
}

gofuncy.Go(ctx, "logged-task", fn, gofuncy.WithMiddleware(logging))
```

## Telemetry

OpenTelemetry tracing and metrics are enabled by default. Every `Go` and `Group.Add` call creates a span and emits metrics.

### Metrics (enabled by default)

| Metric | Type | Description |
|--------|------|-------------|
| `gofuncy.goroutines.started` | Counter | Total goroutines started |
| `gofuncy.goroutines.errors` | Counter | Total goroutine errors |
| `gofuncy.goroutines.active` | UpDownCounter | Currently active goroutines |

### Optional Metrics

| Metric | Type | Enabled via |
|--------|------|-------------|
| `gofuncy.goroutines.duration.seconds` | Histogram | `WithDurationHistogram()` |
| `gofuncy.groups.duration.seconds` | Histogram | `WithDurationHistogram()` on group |

### Disabling Telemetry

```go
gofuncy.Go(ctx, "hot-path", fn,
    gofuncy.WithoutTracing(),
    gofuncy.WithoutStartedCounter(),
    gofuncy.WithoutErrorCounter(),
    gofuncy.WithoutActiveUpDownCounter(),
)
```

### Custom Providers

```go
gofuncy.Go(ctx, "custom-providers", fn,
    gofuncy.WithMeterProvider(customMeterProvider),
    gofuncy.WithTracerProvider(customTracerProvider),
)
```

All metrics use the scope name `github.com/foomo/gofuncy` and the OpenTelemetry schema URL `https://opentelemetry.io/schemas/v1.40.0`.
