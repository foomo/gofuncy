---
prev:
  text: Home
  link: /
next:
  text: Getting Started
  link: /guide/getting-started
---

# Introduction

## The Problem

Go makes concurrency easy to start but hard to get right. A bare `go func()` call has several pitfalls:

- **Unrecovered panics** -- a panic in a goroutine crashes the entire process with no chance to log or handle it gracefully.
- **Lost errors** -- errors returned inside goroutines are silently discarded unless you wire up channels or error groups yourself.
- **No observability** -- you get no metrics, traces, or structured logs about goroutine lifecycle unless you instrument every call site manually.
- **No concurrency control** -- it is easy to accidentally spawn thousands of goroutines with no backpressure.
- **Boilerplate** -- every call site ends up reimplementing the same patterns: wait groups, error collection, context cancellation, panic recovery.

## Why gofuncy?

Every production goroutine eventually needs panic recovery, error handling, and some form of observability. You can bolt these on one by one at each call site, or you can get them all from a single function call.

**Safety by default.** A bare `go func()` that panics takes down your entire process. gofuncy recovers every panic automatically, wraps it in a `*PanicError` with the full stack trace, and routes it through your error handling pipeline.

**Observability without effort.** Every goroutine gets an OpenTelemetry span, started/error/active counters, and optional duration histograms -- out of the box. No manual instrumentation at each call site.

**Composable resilience.** Retry, circuit breaker, fallback, and timeout are built-in options that compose in the correct order. You don't need to import three libraries and wire them together.

**Performance overhead is negligible.** With default telemetry (tracing + counters), gofuncy adds ~1-2μs per call. With telemetry disabled, overhead drops to ~120ns. For any goroutine that does real work (I/O, computation), this cost is invisible -- and it pays for itself the first time you debug a production issue using the traces and metrics you got for free.

## What gofuncy Does

gofuncy provides a small set of functions that wrap common goroutine patterns with built-in safety, observability, and composability:

| Function | Purpose |
|----------|---------|
| [`Go`](/api/go) | Fire-and-forget goroutine with panic recovery and error handling |
| [`Start`](/api/start) | Like `Go`, but blocks until the goroutine is running |
| [`StartWithReady`](/api/start) | Like `Go`, but blocks until the goroutine signals readiness |
| [`StartWithStop`](/api/start) | Like `Go`, but the goroutine receives a stop function to cancel itself |
| [`GoWithCancel`](/api/start) | Like `Go`, but returns a stop function to shut down the goroutine |
| [`Wait`](/api/wait) | Spawn a goroutine and collect the result later via a wait function |
| [`WaitWithStop`](/api/wait) | Like `Wait`, but the goroutine receives a stop function to cancel itself |
| [`WaitWithReady`](/api/wait) | Like `Wait`, but blocks until the goroutine signals readiness |
| [`Do`](/api/do) | Synchronous execution with the full middleware chain (no goroutine) |
| [`NewGroup`](/api/group) | Managed set of concurrent functions with shared lifecycle |
| [`All`](/api/all) | Concurrent iteration over a slice |
| [`Map`](/api/map) | Concurrent transformation of a slice, preserving order |

The [`channel`](/api/channel) subpackage provides an observable generic channel wrapper with built-in metrics for queue depth, in-flight messages, and backpressure detection.

Every function supports a rich [options pattern](/api/options) for configuring behavior per call.

## Design Goals

**Context-first** -- every function takes a `context.Context` as its first parameter. Cancellation, timeouts, and value propagation work the way you expect.

**Composable options** -- configure behavior with functional options like `WithName`, `WithTimeout`, `WithLimit`, and `WithMiddleware`. Options are additive and can be shared or overridden per call.

**Observable by default** -- OpenTelemetry tracing and metrics are enabled out of the box. Every goroutine gets a span, started/error/active counters, and optional duration histograms. Disable what you don't need with `WithoutTracing`, `WithoutStartedCounter`, etc.

**Panic-safe** -- every goroutine spawned by gofuncy recovers from panics automatically. Panics are converted to `*PanicError` with the full stack trace preserved.

**Minimal dependencies** -- gofuncy depends only on the OpenTelemetry SDK and `golang.org/x/sync` for the weighted semaphore. No frameworks, no magic.
