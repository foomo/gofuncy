package channel_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/foomo/gofuncy/channel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleNew() {
	ch := channel.New[string]("events", channel.WithBuffer[string](3))
	defer ch.Close()

	_ = ch.Send(context.Background(), "hello", "world")

	fmt.Println(<-ch.Receive())
	fmt.Println(<-ch.Receive())
	// Output:
	// hello
	// world
}

func TestChannel_sendAndReceive(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("test", channel.WithBuffer[int](1))

	require.NoError(t, ch.Send(t.Context(), 42))

	val, ok := <-ch.Receive()
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	ch.Close()
}

func TestChannel_sendMultipleValues(t *testing.T) {
	t.Parallel()

	ch := channel.New[string]("multi", channel.WithBuffer[string](3))

	require.NoError(t, ch.Send(t.Context(), "a", "b", "c"))

	assert.Equal(t, 3, ch.Len())

	for _, expected := range []string{"a", "b", "c"} {
		val, ok := <-ch.Receive()
		assert.True(t, ok)
		assert.Equal(t, expected, val)
	}

	ch.Close()
}

func TestChannel_unbufferedSendReceive(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("unbuffered")

	var received int

	var wg sync.WaitGroup
	wg.Go(func() {
		val, ok := <-ch.Receive()
		assert.True(t, ok)

		received = val
	})

	require.NoError(t, ch.Send(t.Context(), 99))
	wg.Wait()

	assert.Equal(t, 99, received)

	ch.Close()
}

func TestChannel_rangeReceive(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("range", channel.WithBuffer[int](3))

	require.NoError(t, ch.Send(t.Context(), 1, 2, 3))
	ch.Close()

	var got []int
	for v := range ch.Receive() {
		got = append(got, v)
	}

	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestChannel_sendOnClosedChannel(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("closed")
	ch.Close()

	err := ch.Send(t.Context(), 1)
	require.ErrorIs(t, err, channel.ErrClosed)
}

func TestChannel_doubleCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("double-close")

	ch.Close()
	assert.NotPanics(t, func() {
		ch.Close()
	})
}

func TestChannel_sendContextCancelled(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("ctx-cancel")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := ch.Send(ctx, 1)
	require.ErrorIs(t, err, context.Canceled)

	ch.Close()
}

func TestChannel_sendContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("ctx-deadline")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	err := ch.Send(ctx, 1)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	ch.Close()
}

func TestChannel_concurrentSendAndClose(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("concurrent-close", channel.WithBuffer[int](100))

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 10 {
				err := ch.Send(t.Context(), 1)
				if err != nil {
					assert.ErrorIs(t, err, channel.ErrClosed)
					return
				}
			}
		})
	}

	time.Sleep(5 * time.Millisecond)
	ch.Close()

	wg.Wait()
}

func TestChannel_lenAndCap(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("len-cap", channel.WithBuffer[int](10))

	assert.Equal(t, 0, ch.Len())
	assert.Equal(t, 10, ch.Cap())

	require.NoError(t, ch.Send(t.Context(), 1, 2, 3))

	assert.Equal(t, 3, ch.Len())
	assert.Equal(t, 10, ch.Cap())

	ch.Close()
}

func TestChannel_name(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("my-channel")
	assert.Equal(t, "my-channel", ch.Name())

	ch.Close()
}

func TestChannel_withoutChansCounter(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("no-chans",
		channel.WithBuffer[int](5),
		channel.WithoutChansCounter[int](),
	)

	require.NoError(t, ch.Send(t.Context(), 1))

	<-ch.Receive()
	ch.Close()
}

func TestChannel_withoutMessagesCounter(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("no-messages",
		channel.WithBuffer[int](5),
		channel.WithoutMessagesSentCounter[int](),
	)

	require.NoError(t, ch.Send(t.Context(), 1))

	<-ch.Receive()
	ch.Close()
}

func TestChannel_withDurationHistogram(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("with-duration",
		channel.WithBuffer[int](5),
		channel.WithDurationHistogram[int](),
	)

	require.NoError(t, ch.Send(t.Context(), 1))

	<-ch.Receive()
	ch.Close()
}

func TestChannel_withTracing(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("with-tracing",
		channel.WithBuffer[int](5),
		channel.WithTracing[int](),
	)

	require.NoError(t, ch.Send(t.Context(), 1))

	<-ch.Receive()
	ch.Close()
}

func TestChannel_allTelemetryDisabled(t *testing.T) {
	t.Parallel()

	ch := channel.New[int]("bare",
		channel.WithBuffer[int](5),
		channel.WithoutChansCounter[int](),
		channel.WithoutMessagesSentCounter[int](),
	)

	require.NoError(t, ch.Send(t.Context(), 1, 2, 3))

	assert.Equal(t, 3, ch.Len())

	for range 3 {
		<-ch.Receive()
	}

	ch.Close()
}

// ------------------------------------------------------------------------------------------------
// ~ Benchmarks
// ------------------------------------------------------------------------------------------------

func BenchmarkChannel_rawChan(b *testing.B) {
	ch := make(chan int, 1)

	for b.Loop() {
		ch <- 1

		<-ch
	}
}

func BenchmarkChannel_noTelemetry(b *testing.B) {
	ch := channel.New[int]("bench-bare",
		channel.WithBuffer[int](1),
		channel.WithoutChansCounter[int](),
		channel.WithoutMessagesSentCounter[int](),
	)
	ctx := context.Background()

	b.Cleanup(func() {
		ch.Close()
	})

	for b.Loop() {
		_ = ch.Send(ctx, 1)

		<-ch.Receive()
	}
}

func BenchmarkChannel_defaultTelemetry(b *testing.B) {
	ch := channel.New[int]("bench-default",
		channel.WithBuffer[int](1),
	)
	ctx := context.Background()

	b.Cleanup(func() {
		ch.Close()
	})

	for b.Loop() {
		_ = ch.Send(ctx, 1)

		<-ch.Receive()
	}
}

func BenchmarkChannel_allFeatures(b *testing.B) {
	ch := channel.New[int]("bench-all",
		channel.WithBuffer[int](1),
		channel.WithDurationHistogram[int](),
		channel.WithTracing[int](),
	)
	ctx := context.Background()

	b.Cleanup(func() {
		ch.Close()
	})

	for b.Loop() {
		_ = ch.Send(ctx, 1)

		<-ch.Receive()
	}
}
