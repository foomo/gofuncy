# gofuncy

[![Build Status](https://github.com/foomo/gofuncy/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/gofuncy/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/gofuncy)](https://goreportcard.com/report/github.com/foomo/gofuncy)
[![godoc](https://godoc.org/github.com/foomo/gofuncy?status.svg)](https://godoc.org/github.com/foomo/gofuncy)
[![goreleaser](https://github.com/foomo/gofuncy/actions/workflows/release.yml/badge.svg)](https://github.com/foomo/gofuncy/actions)

> Stop using `go func`, start using `gofuncy`!

- ctx as a first class citizen
- error return as a first class citizen
- optional: enable telemetry (metrics & traces)
  - `gofuncy.routine.count` counter
  - `gofuncy.routine.duration` histogram
  - `gofuncy.channel.sent.count` counter
  - `gofuncy.channel.sent.duration` histogram

## Configuration

Environment variables:

- `OTEL_ENABLED`: enable telemetry
- `GOFUNCY_CHANNEL_VALUE_EVENTS_ENABLED`: creates a span event for every value sent into the channel
- `GOFUNCY_CHANNEL_VALUE_ATTRIBUTE_ENABLED`: adds the json dump of the data to the span event

## Usage

From:

```go
package main

func main() {
  go func() {
    numbers, err := GenerateNumbers(5)
    if err != nil {
      panic(err)
    }
  }()
}
```

To:

```go
package main

import (
  "github.com/foomo/gofuncy"
)

func main() {
  errChan := gofuncy.Go(func(ctx context.Context) error {
    numbers, err := GenerateNumbers(5)
    return err
  })
  if err := <-errChan; err != nil {
    panic(err)
  }
}
```

## Concept

### Routines

#### Error

Each routine can return an error that is being returned through an error channel:

```go
errChan := gofuncy.Go(func (ctx context.Context) error {
return nil
})

if err := <- errChan; err != nil {
panic(err)
}
```

#### Context

Each routine will receive its own base context, which can be set:

```go
errChan := gofuncy.Go(send(msg), gofuncy.WithContext(context.Background()))
```

```mermaid
flowchart TB
  subgraph root
    channel[Channel]
    subgraph "Routine A"
      ctxA[ctx] --> senderA
      senderA[Sender]
    end
    subgraph "Routine B"
      ctxB[ctx] --> senderB
      senderB[Sender]
    end
    senderA --> channel
    senderB --> channel
    channel --> receiverC
    subgraph "Routine C"
      ctxC[ctx] --> receiverC
      receiverC[Receiver]
    end
  end
```

#### Names

Using the context we will inject a name for the process so that it can always be identified:

```mermaid
flowchart TB
  subgraph root
    channel[Channel]
    subgraph "Routine A"
      ctxA[ctx] -- ctx: sender - a --> senderA
      senderA[Sender]
    end
    subgraph "Routine B"
      ctxB[ctx] -- ctx: sender - b --> senderB
      senderB[Sender]
    end
    senderA --> channel
    senderB --> channel
    channel --> receiverC
    subgraph "Routine C"
      ctxC[ctx] -- ctx: receiver - b --> receiverC
      receiverC[Receiver]
    end
  end
```

#### Telemetry

Metrics:

| Name                       | Type          |
|----------------------------|---------------|
| `gofuncy.routine.count`    | UpDownCounter |
| `gofuncy.routine.duration` | Histogram     |

```mermaid
flowchart TB
  subgraph root
    subgraph rA ["Routine A"]
      handler[Handler]
    end
    rA -- gofuncy . routine . count --> Metrics
    rA -- gofuncy . routine . duration --> Metrics
    rA -- span: routine - a --> Trace
  end
```

## How to Contribute

Please refer to the [CONTRIBUTING](.gihub/CONTRIBUTING.md) details and follow the [CODE_OF_CONDUCT](.gihub/CODE_OF_CONDUCT.md) and [SECURITY](.github/SECURITY.md) guidelines.

## License

Distributed under MIT License, please see license file within the code for more details.

_Made with ♥ [foomo](https://www.foomo.org) by [bestbytes](https://www.bestbytes.com)_
