---
prev:
  text: Map
  link: /api/map
next:
  text: Channel
  link: /api/channel
---

# Options Reference

Options are configured via functional option constructors. Each function's API page documents which options it accepts. This page provides a consolidated overview.

## Option Categories

| Category | Interface | Accepted by |
|----------|-----------|-------------|
| Shared | `GoOption` + `GroupOption` | `Do`, `Wait`, `WaitWithStop`, `WaitWithReady`, `Go`, `Start`, `StartWithReady`, `StartWithStop`, `GoWithCancel`, `NewGroup`, `Group.Add` |
| Go-only | `GoOption` | `Do`, `Wait`, `WaitWithStop`, `WaitWithReady`, `Go`, `Start`, `StartWithReady`, `StartWithStop`, `GoWithCancel`, `Group.Add` |
| Group-only | `GroupOption` | `NewGroup`, `All`, `Map` |

The compiler enforces these constraints. You cannot pass a `groupOnlyOpt` to `Go()` or a `goOnlyOpt` to `NewGroup()`.

## Defaults

When no options are passed, the following defaults apply:

| Setting | Default |
|---------|---------|
| Name | Function-specific (e.g. `"gofuncy.go"`, `"gofuncy.group"`) |
| Tracing | Enabled |
| Started counter | Enabled |
| Error counter | Enabled |
| Active counter | Enabled |
| Duration histogram | Disabled |
| Error handler (`Go`/`Start`/`StartWithReady`/`StartWithStop`/`GoWithCancel` only) | `slog.ErrorContext` |
| Limit | No limit |
| Fail-fast | Disabled |
| Retry | Disabled |
| Circuit breaker | Disabled |
| Fallback | Disabled |

## Resilience Chain Order

The framework applies resilience options in a fixed order, regardless of the order they appear in your code:

| Position | Middleware | Behavior |
|----------|-----------|----------|
| Innermost | **Timeout** | Each invocation gets a fresh deadline |
| ↑ | **Retry** | Retries the timeout-wrapped function |
| ↑ | **Circuit Breaker** | Sees the final outcome after all retries |
| Outermost | **Fallback** | Last resort -- catches everything |

For custom ordering, use `WithMiddleware` with the middleware constructors (`Retry()`, `Fallback()`, etc.) directly.

## Option Merging in Group.Add

When you pass options to `Group.Add`, they are merged on top of the group's options using these rules:

| Field type | Merge rule |
|-----------|------------|
| `string` (name) | Override if non-empty |
| `*slog.Logger` | Override if non-nil |
| `[]Middleware` | Append |
| `time.Duration` | Override if > 0 |
| `bool` (metrics/tracing) | OR (enable, never disable) |
| `MeterProvider` / `TracerProvider` | Override if non-nil |
| `*semaphore.Weighted` | Override if non-nil |
| `*CircuitBreaker` | Override if non-nil |
| Retry / Fallback | Override if set |
| `limit`, `failFast` | Not merged (group-only) |
