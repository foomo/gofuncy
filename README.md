[![Build Status](https://github.com/foomo/gofuncy/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/gofuncy/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/gofuncy)](https://goreportcard.com/report/github.com/foomo/gofuncy)
[![GoDoc](https://godoc.org/github.com/foomo/gofuncy?status.svg)](https://godoc.org/github.com/foomo/gofuncy)

<p align="center">
  <img alt="gofuncy" src="docs/public/logo.png" width="400" height="400"/>
</p>

# gofuncy

> Stop using `go func`, start using `gofuncy`!

Context-aware, observable goroutine management with built-in resilience patterns.

## Features

- Context propagation with routine name and parent chain
- Automatic panic recovery
- Built-in telemetry (metrics and tracing via OpenTelemetry)
- Resilience: retry with exponential backoff, circuit breaker, fallback
- Concurrency control via semaphores and group limits
- Stall detection

## Installation

```bash
go get github.com/foomo/gofuncy
```

## Quick Start

```go
ctx := gofuncy.Ctx(context.Background()).Root()

// Fire-and-forget goroutine
gofuncy.Go(ctx, "worker", func(ctx context.Context) error {
    // gofuncy.Ctx(ctx).Name() == "worker"
    return doWork(ctx)
})

// Synchronous execution with middleware chain
err := gofuncy.Do(ctx, "fetch", fetchData)

// Goroutine with wait
wait := gofuncy.Wait(ctx, "processor", processItems)
// ... do other work ...
err := wait()
```

## Core API

Every function wraps a `gofuncy.Func`:

```go
type Func func(ctx context.Context) error
```

| Function | Description |
|----------|-------------|
| `Go(ctx, name, fn, ...GoOption)` | Fire-and-forget goroutine with error logging |
| `Do(ctx, name, fn, ...GoOption)` | Synchronous execution, returns error directly |
| `Wait(ctx, name, fn, ...GoOption)` | Goroutine that returns a wait function |
| `NewGroup(ctx, name, ...GroupOption)` | Concurrent group with shared lifecycle |
| `All(ctx, name, items, fn, ...GroupOption)` | Execute fn for each item concurrently |
| `Map(ctx, name, items, fn, ...GroupOption)` | Transform items concurrently, preserving order |

## Options

```go
// Resilience
gofuncy.WithTimeout(5 * time.Second)
gofuncy.WithRetry(3)
gofuncy.WithCircuitBreaker(cb)
gofuncy.WithFallback(fallbackFn)

// Concurrency
gofuncy.WithLimit(10)      // Group only
gofuncy.WithLimiter(sem)   // Shared semaphore

// Telemetry (on by default, opt-out)
gofuncy.WithoutTracing()
gofuncy.WithoutStartedCounter()
gofuncy.WithoutErrorCounter()
gofuncy.WithoutActiveUpDownCounter()
gofuncy.WithDurationHistogram() // opt-in
```

## Telemetry

Metrics (all via OpenTelemetry):

| Name | Type | Default |
|------|------|---------|
| `gofuncy.goroutines.started` | Counter | on |
| `gofuncy.goroutines.errors` | Counter | on |
| `gofuncy.goroutines.active` | UpDownCounter | on |
| `gofuncy.goroutines.stalled` | Counter | on |
| `gofuncy.goroutines.duration.seconds` | Histogram | off |
| `gofuncy.groups.duration.seconds` | Histogram | off |

## Channel

The `channel` subpackage provides a generic, observable channel:

```go
import "github.com/foomo/gofuncy/channel"

ch := channel.New[string]("events", channel.WithBuffer[string](100))
defer ch.Close()

ch.Send(ctx, "hello", "world")

for msg := range ch.Receive() {
    fmt.Println(msg)
}
```

Channel metrics:

| Name | Type | Default |
|------|------|---------|
| `gofuncy.chans.current` | UpDownCounter | on |
| `gofuncy.messages.sent` | Counter | on |
| `gofuncy.messages.duration.seconds` | Histogram | off |

## How to Contribute

Contributions are welcome! Please read the [contributing guide](docs/CONTRIBUTING.md).

![Contributors](https://contributors-table.vercel.app/image?repo=foomo/gofuncy&width=50&columns=15)

## License

Distributed under MIT License, please see the [license](LICENSE) file within the code for more details.

_Made with ♥ [foomo](https://www.foomo.org) by [bestbytes](https://www.bestbytes.com)_
