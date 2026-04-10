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

- **Lost errors** -- errors returned inside goroutines are silently discarded unless you wire up channels or error groups yourself.
- **Unrecovered panics** -- a panic in a goroutine crashes the entire process with no chance to log or handle it gracefully.
- **No observability** -- you get no metrics, traces, or structured logs about goroutine lifecycle unless you instrument every call site manually.
- **No concurrency control** -- it is easy to accidentally spawn thousands of goroutines with no backpressure.
- **Boilerplate** -- every call site ends up reimplementing the same patterns: wait groups, error collection, context cancellation, panic recovery.

## What gofuncy Does

gofuncy provides a small set of functions that wrap common goroutine patterns with built-in safety, observability, and composability:

| Function | Purpose |
|----------|---------|
| [`Go`](/api/go) | Fire-and-forget goroutine with panic recovery and error handling |
| [`Wait`](/api/wait) | Spawn a goroutine and collect the result later via a wait function |
| [`Do`](/api/do) | Synchronous execution with the full middleware chain (no goroutine) |
| [`NewGroup`](/api/group) | Managed set of concurrent functions with shared lifecycle |
| [`All`](/api/all) | Concurrent iteration over a slice |
| [`Map`](/api/map) | Concurrent transformation of a slice, preserving order |

The [`channel`](/api/channel) subpackage provides an observable generic channel wrapper with built-in metrics for queue depth, in-flight messages, and backpressure detection.

Every function supports a rich [options pattern](/api/options) for configuring behavior per call.

## Design Goals

**Context-first** -- every function takes a `context.Context` as its first parameter. Cancellation, timeouts, and value propagation work the way you expect.

**Composable options** -- configure behavior with functional options like `WithTimeout`, `WithLimit`, and `WithMiddleware`. The `name` parameter is always required for clear observability. Options are additive and can be shared or overridden per call.

**Observable by default** -- OpenTelemetry tracing and metrics are enabled out of the box. Every goroutine gets a span, started/error/active counters, and optional duration histograms. Disable what you don't need with `WithoutTracing`, `WithoutStartedCounter`, etc.

**Panic-safe** -- every goroutine spawned by gofuncy recovers from panics automatically. Panics are converted to `*PanicError` with the full stack trace preserved.

**Minimal dependencies** -- gofuncy depends only on the OpenTelemetry SDK and `golang.org/x/sync` for the weighted semaphore. No frameworks, no magic.
