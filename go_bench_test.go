package gofuncy_test

import (
	"context"
	"testing"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func BenchmarkGoRaw(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		errChan := make(chan error, 1)

		go func() {
			errChan <- nil

			close(errChan)
		}()

		<-errChan
	}
}

func BenchmarkGo(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, func(ctx context.Context) error {
			return nil
		})
		<-errChan
	}
}

func BenchmarkGo_withName(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx,
			func(ctx context.Context) error {
				return nil
			},
			gofuncy.WithName("benchmark-routine"),
		)
		<-errChan
	}
}

func BenchmarkGo_withTelemetry(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()
	meterProvider := noop.NewMeterProvider()
	tracerProvider := tracenoop.NewTracerProvider()

	for b.Loop() {
		errChan := gofuncy.Go(ctx,
			func(ctx context.Context) error {
				return nil
			},
			gofuncy.WithMeter(meterProvider.Meter("bench")),
			gofuncy.WithTracer(tracerProvider.Tracer("bench")),
			gofuncy.WithDurationMetricEnabled(true),
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
	meterProvider := noop.NewMeterProvider()
	tracerProvider := tracenoop.NewTracerProvider()

	for b.Loop() {
		c := gofuncy.NewChan[int](
			gofuncy.ChanWithBuffer[int](1),
			gofuncy.ChanWithMeter[int](meterProvider.Meter("bench")),
			gofuncy.ChanWithTracer[int](tracerProvider.Tracer("bench")),
			gofuncy.ChanWithMessagesDurationMetricEnabled[int](true),
		)
		if err := c.Send(ctx, 42); err != nil {
			b.Fatal(err)
		}

		<-c.Receive(ctx)
		c.Close()
	}
}
