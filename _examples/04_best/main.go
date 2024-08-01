package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

var meter *metric.MeterProvider
var tracer *trace.TracerProvider

func init() {
	{
		exp, _ := stdoutmetric.New(
			stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithWriter(os.Stdout),
		)
		meter = metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(exp)),
		)
		otel.SetMeterProvider(meter)
	}
	{
		exp, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
		tracer = trace.NewTracerProvider(
			trace.WithBatcher(exp),
		)
		otel.SetTracerProvider(tracer)
	}
	{
		exp, err := stdoutlog.New()
		if err != nil {
			panic(err)
		}

		processor := log.NewSimpleProcessor(exp)
		provider := log.NewLoggerProvider(log.WithProcessor(processor))
		defer func() {
			if err := provider.Shutdown(context.Background()); err != nil {
				panic(err)
			}
		}()

		global.SetLoggerProvider(provider)
	}
}

func main() {
	go func() {
		time.Sleep(10 * time.Second)
		// _ = meter.ForceFlush(context.Background())
		_ = tracer.ForceFlush(context.Background())
		os.Exit(0)
	}()

	msg := gofuncy.NewChannel[string](
		// gofuncy.ChannelWithBufferSize[string](5),
		gofuncy.ChannelWithTelemetryEnabled[string](true),
		gofuncy.ChannelWithValueEventsEnabled[string](true),
		gofuncy.ChannelWithValueAttributeEnabled[string](true),
	)

	fmt.Println("start")

	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-a"), gofuncy.WithTelemetryEnabled(true))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-b"), gofuncy.WithTelemetryEnabled(true))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-c"), gofuncy.WithTelemetryEnabled(true))

	_ = gofuncy.Go(receive(msg), gofuncy.WithName("receiver-a"))
	_ = receive(msg)(context.Background())

}

func send(msg *gofuncy.Channel[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			if err := msg.Send(ctx, fmt.Sprintf("Hello World")); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func receive(msg *gofuncy.Channel[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for m := range msg.Receive() {
			fmt.Println("Message:", m.Data, "Handler:", gofuncy.RoutineFromContext(ctx), "Sender:", gofuncy.SenderFromContext(m.Context()))
			// fmt.Println(m, len(msg))
			time.Sleep(time.Second)
		}
		return nil
	}
}
