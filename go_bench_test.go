package gofuncy_test

import (
	"context"
	"testing"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func BenchmarkGoRaw(b *testing.B) {
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
	ctx := gofuncy.Ctx(context.Background()).Root()
	for b.Loop() {
		errChan := gofuncy.Go(ctx, func(ctx context.Context) error {
			return nil
		})
		<-errChan
	}
}

func BenchmarkGo_withName(b *testing.B) {
	ctx := gofuncy.Ctx(context.Background()).Root()
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
	ctx := gofuncy.Ctx(context.Background()).Root()
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
