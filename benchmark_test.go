package gofuncy_test

import (
	"context"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossymetric"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
)

var gofunc = func(ctx context.Context) error {
	time.Sleep(time.Millisecond)
	return nil
}

func BenchmarkGoRaw(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		errChan := make(chan error, 1)

		go func() {
			errChan <- gofunc(b.Context())

			close(errChan)
		}()

		<-errChan
	}
}

func BenchmarkAsync(b *testing.B) {
	b.ReportAllocs()

	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc)
		<-errChan
	}
}

func BenchmarkAsync_withName(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc,
			gofuncy.AsyncOption().WithName("benchmark-routine"),
		)
		<-errChan
	}
}

func BenchmarkAsync_withTracing(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc,
			gofuncy.AsyncOption().WithTracing(),
		)
		<-errChan
	}
}

func BenchmarkAsync_withCounterMetric(b *testing.B) {
	b.ReportAllocs()
	oteltesting.ReportMetrics(b, glossymetric.NewTest(b))

	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc,
			gofuncy.AsyncOption().WithCounterMetric(),
		)
		<-errChan
	}
}

func BenchmarkAsync_withUpDownMetric(b *testing.B) {
	b.ReportAllocs()
	oteltesting.ReportMetrics(b, glossymetric.NewTest(b))

	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc,
			gofuncy.AsyncOption().WithUpDownMetric(),
		)
		<-errChan
	}
}

func BenchmarkAsync_withDurationMetric(b *testing.B) {
	b.ReportAllocs()
	oteltesting.ReportMetrics(b, glossymetric.NewTest(b))

	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Async(ctx, gofunc,
			gofuncy.AsyncOption().WithDurationMetric(),
		)
		<-errChan
	}
}

func BenchmarkNewChan(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		c := gofuncy.NewChan[int]()
		c.Close()
	}
}

func BenchmarkChan_SendReceive(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		c := gofuncy.NewChan[int](gofuncy.ChanWithBuffer[int](1))

		if err := c.Send(ctx, 42); err != nil {
			b.Fatal(err)
		}

		<-c.Receive(ctx)
		c.Close()
	}
}

func BenchmarkChan_ReceiveValue(b *testing.B) {
	b.ReportAllocs()

	ctx := gofuncy.Ctx(b.Context()).Root()
	for b.Loop() {
		c := gofuncy.NewChan[int](gofuncy.ChanWithBuffer[int](1))
		if err := c.Send(ctx, 42); err != nil {
			b.Fatal(err)
		}

		_, _ = c.ReceiveValue(ctx)
		c.Close()
	}
}

func BenchmarkChan_withTelemetry(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		c := gofuncy.NewChan[int](
			gofuncy.ChanWithBuffer[int](1),
			gofuncy.ChanWithTracing[int](),
			gofuncy.ChanWithCounterMetric[int](),
			gofuncy.ChanWithMessagesDurationMetric[int](),
		)
		if err := c.Send(ctx, 42); err != nil {
			b.Fatal(err)
		}

		<-c.Receive(ctx)
		c.Close()
	}
}
