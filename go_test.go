package gofuncy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func ExampleGo() {
	done := make(chan struct{})

	gofuncy.Go(context.Background(), func(ctx context.Context) error {
		defer close(done)

		fmt.Println("running")

		return nil
	})

	<-done
	// Output:
	// running
}

func TestGo_basic(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withTracing(t *testing.T) {
	t.Parallel()
	ReportTraces(t)

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithTracing(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withCounterMetric(t *testing.T) {
	t.Parallel()

	ReportMetrics(t)

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithCounterMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withUpDownMetric(t *testing.T) {
	t.Parallel()

	ReportMetrics(t)

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithUpDownMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withDurationMetric(t *testing.T) {
	t.Parallel()

	ReportMetrics(t)

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithDurationMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_errorHandler(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			return fmt.Errorf("test error")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.EqualError(t, err, "test error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error handler")
	}
}

func TestGo_panicRecovery(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			panic("fire and forget panic")
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		var panicErr *gofuncy.PanicError
		require.ErrorAs(t, err, &panicErr)
		assert.Equal(t, "fire and forget panic", panicErr.Value)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestGo_contextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			return nil
		},
		gofuncy.WithErrorHandler(func(ctx context.Context, err error) {
			errCh <- err
		}),
	)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for context error")
	}
}
