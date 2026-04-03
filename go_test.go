package gofuncy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	testingx "github.com/foomo/go/testing"
	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossymetric"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type RunFunc func() int

func M(m *testing.M, fn func(m *testing.M) RunFunc) RunFunc {
	return fn(m)
}

func (r RunFunc) Run() int {

	return r()
}

func TestMain(m *testing.M) {
	_, flush := oteltesting.TestMainReportMetrics(m, glossymetric.NewTestMain(m))
	goleak.VerifyTestMain(testingx.MFunc(func() int {
		i := m.Run()
		flush()
		return i
	}))
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
	oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.GoOption().WithTracing(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withCounterMetric(t *testing.T) {
	t.Parallel()
	// _, flush := oteltesting.ReportMetrics(t, glossymetric.NewTesting(t))
	// defer flush()

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.GoOption().WithCounterMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withUpDownMetric(t *testing.T) {
	t.Parallel()
	// _, flush := oteltesting.ReportMetrics(t, glossymetric.NewTesting(t))
	// defer flush()

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.GoOption().WithUpDownMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_withDurationMetric(t *testing.T) {
	t.Parallel()
	// _, flush := oteltesting.ReportMetrics(t, glossymetric.NewTesting(t))
	// defer flush()

	done := make(chan struct{})

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.GoOption().WithDurationMetric(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Go to complete")
	}
}

func TestGo_errorHandler(t *testing.T) {
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			return fmt.Errorf("test error")
		},
		gofuncy.GoOption().WithErrorHandler(func(ctx context.Context, err error) {
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
	errCh := make(chan error, 1)

	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			panic("fire and forget panic")
		},
		gofuncy.GoOption().WithErrorHandler(func(ctx context.Context, err error) {
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

func TestGo_canceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	errCh := make(chan error, 1)

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			return nil
		},
		gofuncy.GoOption().WithErrorHandler(func(ctx context.Context, err error) {
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

func TestGo_contextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	errCh := make(chan error, 1)

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			cancel()
			return nil
		},
		gofuncy.GoOption().WithErrorHandler(func(ctx context.Context, err error) {
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
