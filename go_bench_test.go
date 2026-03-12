package gofuncy_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

func BenchmarkGo(b *testing.B) {
	b.ReportAllocs()

	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, gofunc)
		<-errChan
	}
}

type xx struct {
	m *testing.M
}

func Run(m *testing.M) xx {
	return xx{m: m}
}

func (x xx) Run() int {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	otel.SetMeterProvider(meterProvider)

	traceExporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(traceExporter),
	)

	otel.SetTracerProvider(tracerProvider)

	rm := &metricdata.ResourceMetrics{}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := tracerProvider.ForceFlush(ctx); err != nil {
			panic(err)
		}

		spans := traceExporter.GetSpans()
		if len(spans) > 0 {
			fmt.Println("\nOTEL TRACES")
		}

		printScopeTraces(spans)

		err := reader.Collect(ctx, rm)
		if err != nil {
			panic(err)
		}

		if len(rm.ScopeMetrics) > 0 {
			fmt.Println("\nOTEL METRICS")
		}

		printScopeMetrics(rm)

		if err := tracerProvider.Shutdown(ctx); err != nil {
			panic(err)
		}

		if err := meterProvider.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()

	return x.m.Run()
}

func printScopeMetrics(rm *metricdata.ResourceMetrics) {
	const (
		nameWidth = 40
		sep       = "─"
	)

	formatAttrs := func(a attribute.Set) string {
		parts := make([]string, 0, a.Len())
		for _, v := range a.ToSlice() {
			parts = append(parts, fmt.Sprintf("%s=%s", string(v.Key), v.Value.Emit()))
		}
		if len(parts) == 0 {
			return ""
		}
		return " {" + strings.Join(parts, ", ") + "}"
	}

	printHeader := func(name, kind string, attrs attribute.Set) {
		fmt.Printf("\n%-*s [%s]%s\n", nameWidth, name, kind, formatAttrs(attrs))
		fmt.Printf("%s\n", strings.Repeat(sep, nameWidth+20))
	}

	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch agg := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range agg.DataPoints {
					printHeader(m.Name, "counter", dp.Attributes)
					fmt.Printf("  %-12s %d\n", "value:", dp.Value)
				}
			case metricdata.Sum[float64]:
				for _, dp := range agg.DataPoints {
					printHeader(m.Name, "counter", dp.Attributes)
					fmt.Printf("  %-12s %.3f\n", "value:", dp.Value)
				}
			case metricdata.Histogram[float64]:
				for _, dp := range agg.DataPoints {
					printHeader(m.Name, "histogram", dp.Attributes)

					// One-line summary - no unit assumption
					minV, minOk := dp.Min.Value()
					maxV, maxOk := dp.Max.Value()
					avg := 0.0
					if minOk && maxOk {
						avg = (minV + maxV) / 2
					}

					fmt.Printf("Summary: Min %.3f, Max %.3f, Avg %.3f, Sum %.3f, Count %d\n\n",
						minV, maxV, avg, dp.Sum, dp.Count)

					// Buckets table - no unit assumption
					if len(dp.Bounds) > 0 {
						fmt.Println("Buckets:")
						fmt.Println("┌────────────┬──────────┬─────────┐")
						fmt.Println("│ Range      │ Count    │ %       │")
						fmt.Println("├────────────┼──────────┼─────────┤")

						total := dp.Count
						for i, b := range dp.Bounds {
							pct := float64(dp.BucketCounts[i]) / float64(total) * 100
							fmt.Printf("│ %-10.1f │ %-8d │ %6.1f%% │\n", b, dp.BucketCounts[i], pct)
						}
						fmt.Println("└────────────┴──────────┴─────────┘")
					}
					fmt.Println()
				}
			case metricdata.Gauge[int64]:
				for _, dp := range agg.DataPoints {
					printHeader(m.Name, "gauge", dp.Attributes)
					fmt.Printf("  %-12s %d\n", "value:", dp.Value)
				}
			case metricdata.Gauge[float64]:
				for _, dp := range agg.DataPoints {
					printHeader(m.Name, "gauge", dp.Attributes)
					fmt.Printf("  %-12s %.3f\n", "value:", dp.Value)
				}
			}
		}
	}
}

func printScopeTraces(spans tracetest.SpanStubs) {
	const (
		nameWidth = 40
		sep       = "─"
	)

	for _, s := range spans {
		duration := s.EndTime.Sub(s.StartTime)

		fmt.Printf("\n%-*s [%s] %s\n", nameWidth, s.Name, s.Status.Code, duration)
		fmt.Printf("%s\n", strings.Repeat(sep, nameWidth+20))

		printedSomething := false

		if s.SpanKind.String() != "" {
			fmt.Printf("├─ Span Kind: %s\n", s.SpanKind)
			printedSomething = true
		}

		if len(s.Attributes) > 0 {
			fmt.Printf("└─ Attributes:\n")
			for i, a := range s.Attributes {
				prefix := "   "
				if i == 0 {
					prefix = "   ├─ "
				} else if i == len(s.Attributes)-1 {
					prefix = "   └─ "
				} else {
					prefix = "   ├─ "
				}
				fmt.Printf("%s%s: %s\n", prefix, string(a.Key), a.Value.Emit())
			}
			printedSomething = true
		}

		if len(s.Events) > 0 && printedSomething {
			fmt.Println("   ")
			fmt.Printf("   └─ Events:\n")
			for i, e := range s.Events {
				prefix := "      "
				if i == 0 {
					prefix = "      ├─ "
				} else if i == len(s.Events)-1 {
					prefix = "      └─ "
				} else {
					prefix = "      ├─ "
				}
				fmt.Printf("%s%s", prefix, e.Name)

				if len(e.Attributes) > 0 {
					parts := make([]string, 0, len(e.Attributes))
					for _, a := range e.Attributes {
						parts = append(parts, fmt.Sprintf("%s=%s", string(a.Key), a.Value.Emit()))
					}
					fmt.Printf(" {%s}", strings.Join(parts, ", "))
				}
				fmt.Println()
			}
		}

		if printedSomething || len(s.Events) > 0 {
			fmt.Println()
		}
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

		if len(rm.ScopeMetrics) > 0 {
			fmt.Println("\nOTEL METRICS:")
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
		errChan := gofuncy.Go(ctx, gofunc,
			gofuncy.WithName("benchmark-routine"),
		)
		<-errChan
	}
}

func BenchmarkGo_withTracing(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, gofunc,
			gofuncy.WithTracing(),
		)
		<-errChan
	}
}

func BenchmarkGo_withCounterMetric(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, gofunc,
			gofuncy.WithCounterMetric(),
		)
		<-errChan
	}
}

func BenchmarkGo_withDurationMetric(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, gofunc,
			gofuncy.WithDurationMetric(),
		)
		<-errChan
	}
}

func BenchmarkGo_withTelemetry(b *testing.B) {
	b.ReportAllocs()
	ctx := gofuncy.Ctx(b.Context()).Root()

	for b.Loop() {
		errChan := gofuncy.Go(ctx, gofunc,
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
