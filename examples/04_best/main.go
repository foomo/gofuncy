package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

var meterProvider *metric.MeterProvider
var tracerProvider *trace.TracerProvider

func init() {
	{
		exp, _ := stdoutmetric.New(
			stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithWriter(os.Stdout),
		)
		meterProvider = metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(exp)),
		)
		otel.SetMeterProvider(meterProvider)
	}
	{
		exp, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
		tracerProvider = trace.NewTracerProvider(
			trace.WithSampler(trace.AlwaysSample()),
			trace.WithBatcher(exp),
		)
		otel.SetTracerProvider(tracerProvider)
	}
}

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx := gofuncy.Ctx(context.Background()).Root()

	go func() {
		time.Sleep(10 * time.Second)
		_ = meterProvider.ForceFlush(context.Background())
		_ = tracerProvider.ForceFlush(ctx)
		l.Info("exiting")
		os.Exit(0)
	}()

	msg := gofuncy.NewChan[string](
		gofuncy.ChanWithTracing[string](),
		gofuncy.ChanWithCounterMetric[string](),
		gofuncy.ChanWithMessagesAttributeEnabled[string](true),
	)

	l.Info("start")

	gofuncy.Go(ctx, send(msg), gofuncy.GoOption().WithName("sender-a").WithTracing())
	gofuncy.Go(ctx, send(msg), gofuncy.GoOption().WithName("sender-b").WithTracing())
	gofuncy.Go(ctx, send(msg), gofuncy.GoOption().WithName("sender-c").WithTracing())

	gofuncy.Go(ctx, receive(l, msg), gofuncy.GoOption().WithName("receiver-a").WithTracing())

	time.Sleep(time.Minute)
}

func send(msg *gofuncy.Chan[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			if err := msg.Send(ctx, "Hello World"); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func receive(l *slog.Logger, msg *gofuncy.Chan[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for v := range msg.Receive(ctx) {
			l.InfoContext(ctx, "received message",
				"data", v,
				"handler", gofuncy.NameFromContext(ctx),
				"sender", gofuncy.NameFromContext(ctx),
			)
			time.Sleep(time.Second)
		}
		return nil
	}
}
