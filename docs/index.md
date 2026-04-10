---
layout: home

hero:
  name: gofuncy
  text: Structured Concurrency for Go
  tagline: Safe, observable, and composable goroutine primitives. Stop using go func, start using gofuncy.
  image:
    src: /logo.png
    alt: gofuncy
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: API Reference
      link: /api/go

features:
  - title: Type-Safe Generics
    details: All and Map use Go generics for fully typed concurrent iteration and transformation, with order-preserving results.
  - title: Functional Options
    details: Composable options pattern for configuring names, timeouts, middleware, concurrency limits, and telemetry per call.
  - title: OpenTelemetry Built-In
    details: Automatic tracing, started/error/active counters, and duration histograms out of the box. Disable or customize per operation.
  - title: Semaphore Limiting
    details: Control concurrency with per-group WithLimit or a shared weighted semaphore via WithLimiter across multiple call sites.
  - title: Panic Recovery
    details: Every goroutine is wrapped with automatic panic recovery. Panics are converted to PanicError with full stack traces.
  - title: Middleware Support
    details: Wrap any function with custom middleware for logging, retries, circuit breaking, or any cross-cutting concern.
---
