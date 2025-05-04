package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
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
			trace.WithSampler(trace.AlwaysSample()),
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
	l := otelzap.New(zap.NewExample())

	ctx := gofuncy.RootContext(context.Background())

	go func() {
		time.Sleep(10 * time.Second)
		// _ = meter.ForceFlush(context.Background())
		_ = tracer.ForceFlush(ctx)
		l.Info("exiting")
		os.Exit(0)
	}()

	msg := gofuncy.NewChannel[string](
		// gofuncy.ChannelWithBufferSize[string](5),
		gofuncy.ChannelWithTelemetryEnabled[string](true),
		gofuncy.ChannelWithValueEventsEnabled[string](true),
		gofuncy.ChannelWithValueAttributeEnabled[string](true),
	)

	l.Info("start")

	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-a"), gofuncy.WithTelemetryEnabled(true))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-b"), gofuncy.WithTelemetryEnabled(true))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-c"), gofuncy.WithTelemetryEnabled(true))

	_ = gofuncy.Go(receive(l, msg), gofuncy.WithName("receiver-a"), gofuncy.WithTelemetryEnabled(true))
	// _ = receive(l, msg)(ctx)

	time.Sleep(time.Minute)
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

func receive(l *otelzap.Logger, msg *gofuncy.Channel[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for m := range msg.Receive() {
			l.Ctx(ctx).Error("received message",
				zap.String("data", m.Data),
				zap.String("handler", gofuncy.RoutineFromContext(ctx)),
				zap.String("sender", gofuncy.SenderFromContext(m.Context())),
			)
			// fmt.Println(m, len(msg))
			time.Sleep(time.Second)
		}
		return nil
	}
}
