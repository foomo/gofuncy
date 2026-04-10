package channel_test

import (
	"context"
	"testing"

	"github.com/foomo/gofuncy/channel"
)

func BenchmarkChannel_Send(b *testing.B) {
	ch := channel.New[int]("bench",
		channel.WithBuffer[int](1024),
	)
	defer ch.Close()

	ctx := context.Background()

	// drain in background
	go func() {
		for range ch.Receive() {
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = ch.Send(ctx, 1)
	}
}

func BenchmarkChannel_Send_NoTelemetry(b *testing.B) {
	ch := channel.New[int]("bench",
		channel.WithBuffer[int](1024),
		channel.WithoutChansCounter[int](),
		channel.WithoutMessagesSentCounter[int](),
	)
	defer ch.Close()

	ctx := context.Background()

	// drain in background
	go func() {
		for range ch.Receive() {
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = ch.Send(ctx, 1)
	}
}
