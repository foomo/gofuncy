package gofuncy_test

import (
	"context"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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

func setupOtelBenchmark(b *testing.B) {
	b.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	otel.SetMeterProvider(provider)

	rm := &metricdata.ResourceMetrics{}

	b.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(b.Context()), time.Second)
		defer cancel()

		err := reader.Collect(ctx, rm)
		if err != nil {
			b.Fatal(err)
		}

		// Iterate ResourceMetrics → ScopeMetrics → Metrics → DataPoints
		for _, scope := range rm.ScopeMetrics {
			for _, m := range scope.Metrics {
				switch agg := m.Data.(type) {
				case metricdata.Sum[int64]:
					// Sum.DataPoints is []DataPoint[int64]
					for _, dp := range agg.DataPoints {
						b.ReportMetric(float64(dp.Value), m.Name+"/sum")
					}
				case metricdata.Histogram[int64]:
					// Histogram.DataPoints is []HistogramDataPoint[int64]
					for _, dp := range agg.DataPoints {
						b.ReportMetric(float64(dp.Count), m.Name+"/count")
						b.ReportMetric(float64(dp.Sum), m.Name+"/sum")
					}
				case metricdata.Gauge[int64]:
					for _, dp := range agg.DataPoints {
						b.ReportMetric(float64(dp.Value), m.Name+"/gauge")
					}
				}
			}
		}

		if err := provider.Shutdown(ctx); err != nil {
			b.Fatal(err)
		}
	})
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
	setupOtelBenchmark(b)
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx,
			func(ctx context.Context) error {
				time.Sleep(time.Millisecond)
				return nil
			},
			gofuncy.WithTracing(),
			gofuncy.WithCounterMetric(),
			gofuncy.WithDurationMetric(),
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
	setupOtelBenchmark(b)
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
