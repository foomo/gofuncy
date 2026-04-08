package gofuncy_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"golang.org/x/sync/semaphore"
)

var run = func(ctx context.Context) error {
	time.Sleep(time.Millisecond)
	return nil
}

func BenchmarkGoRaw(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		errChan := make(chan error, 1)

		go func() {
			errChan <- run(b.Context())

			close(errChan)
		}()

		<-errChan
	}
}

func BenchmarkGo_Default(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		done := make(chan struct{})

		gofuncy.Go(b.Context(),
			func(ctx context.Context) error {
				time.Sleep(time.Millisecond)
				close(done)

				return nil
			},
		)

		<-done
	}
}

func BenchmarkGo_Minimal(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		done := make(chan struct{})

		gofuncy.Go(b.Context(),
			func(ctx context.Context) error {
				time.Sleep(time.Millisecond)
				close(done)

				return nil
			},
			gofuncy.WithoutTracing(),
			gofuncy.WithoutStartedCounter(),
			gofuncy.WithoutErrorCounter(),
			gofuncy.WithoutActiveUpDownCounter(),
		)

		<-done
	}
}

func BenchmarkGo_WithLimiter(b *testing.B) {
	b.ReportAllocs()

	sem := semaphore.NewWeighted(4)

	for b.Loop() {
		done := make(chan struct{})

		gofuncy.Go(b.Context(),
			func(ctx context.Context) error {
				time.Sleep(time.Millisecond)
				close(done)

				return nil
			},
			gofuncy.WithLimiter(sem),
			gofuncy.WithoutTracing(),
			gofuncy.WithoutStartedCounter(),
			gofuncy.WithoutErrorCounter(),
			gofuncy.WithoutActiveUpDownCounter(),
		)

		<-done
	}
}

func BenchmarkGroupRaw(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		var wg sync.WaitGroup

		for range 100 {
			wg.Go(func() {
				time.Sleep(time.Millisecond)
			})
		}

		wg.Wait()
	}
}

func BenchmarkGroup(b *testing.B) {
	noTelemetry := []gofuncy.GroupOption{
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	}

	for _, size := range []int{5, 100, 10000} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				g := gofuncy.NewGroup(b.Context(), noTelemetry...)

				for range size {
					g.Add(func(ctx context.Context) error {
						return nil
					})
				}

				_ = g.Wait()
			}
		})
	}
}

func BenchmarkGroup_WithTelemetry(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		g := gofuncy.NewGroup(b.Context(),
			gofuncy.WithName("bench-group"),
			gofuncy.WithDurationHistogram(),
		)

		for range 100 {
			g.Add(func(ctx context.Context) error {
				return nil
			})
		}

		_ = g.Wait()
	}
}

func BenchmarkMap(b *testing.B) {
	noTelemetry := []gofuncy.GroupOption{
		gofuncy.WithoutTracing(),
		gofuncy.WithoutStartedCounter(),
		gofuncy.WithoutErrorCounter(),
		gofuncy.WithoutActiveUpDownCounter(),
	}

	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	b.Run("size=100", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_, _ = gofuncy.Map(b.Context(), items, func(ctx context.Context, item int) (int, error) {
				return item * 2, nil
			}, noTelemetry...)
		}
	})
}
