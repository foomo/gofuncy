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

## How to Contribute

Make a pull request...

## License

Distributed under MIT License, please see license file within the code for more details.
