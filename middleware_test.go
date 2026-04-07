package gofuncy

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// ------------------------------------------------------------------------------------------------
// ~ withContextInjection
// ------------------------------------------------------------------------------------------------

func TestWithContextInjection_nameOnly(t *testing.T) {
	t.Parallel()

	fn := withContextInjection(func(ctx context.Context) error {
		assert.Equal(t, "child", NameFromContext(ctx))
		assert.Empty(t, ParentFromContext(ctx))
		return nil
	}, "child")

	require.NoError(t, fn(context.Background()))
}

func TestWithContextInjection_parentOnly(t *testing.T) {
	t.Parallel()

	ctx := injectNameIntoContext(context.Background(), "parent")

	fn := withContextInjection(func(ctx context.Context) error {
		assert.Equal(t, "parent", ParentFromContext(ctx))
		return nil
	}, NameNoName)

	require.NoError(t, fn(ctx))
}

func TestWithContextInjection_nameAndParent(t *testing.T) {
	t.Parallel()

	ctx := injectNameIntoContext(context.Background(), "parent")

	fn := withContextInjection(func(ctx context.Context) error {
		assert.Equal(t, "child", NameFromContext(ctx))
		assert.Equal(t, "parent", ParentFromContext(ctx))
		return nil
	}, "child")

	require.NoError(t, fn(ctx))
}

func TestWithContextInjection_neither(t *testing.T) {
	t.Parallel()

	fn := withContextInjection(func(ctx context.Context) error {
		assert.Equal(t, NameNoName, NameFromContext(ctx))
		return nil
	}, NameNoName)

	require.NoError(t, fn(context.Background()))
}

// ------------------------------------------------------------------------------------------------
// ~ withTimeout
// ------------------------------------------------------------------------------------------------

func TestWithTimeout_completesBeforeDeadline(t *testing.T) {
	t.Parallel()

	fn := withTimeout(func(ctx context.Context) error {
		return nil
	}, time.Second)

	require.NoError(t, fn(context.Background()))
}

func TestWithTimeout_exceedsDeadline(t *testing.T) {
	t.Parallel()

	fn := withTimeout(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, time.Millisecond)

	err := fn(context.Background())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// ------------------------------------------------------------------------------------------------
// ~ withRecover
// ------------------------------------------------------------------------------------------------

func TestWithRecover_noError(t *testing.T) {
	t.Parallel()

	fn := withRecover(func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, fn(context.Background()))
}

func TestWithRecover_errorPassthrough(t *testing.T) {
	t.Parallel()

	fn := withRecover(func(ctx context.Context) error {
		return fmt.Errorf("test error")
	})

	err := fn(context.Background())
	require.EqualError(t, err, "test error")
}

func TestWithRecover_panic(t *testing.T) {
	t.Parallel()

	fn := withRecover(func(ctx context.Context) error {
		panic("boom")
	})

	err := fn(context.Background())
	require.Error(t, err)

	var panicErr *PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "boom", panicErr.Value)
	assert.NotEmpty(t, panicErr.Stack)
}

// ------------------------------------------------------------------------------------------------
// ~ withStartedCounter
// ------------------------------------------------------------------------------------------------

func TestWithStartedCounter(t *testing.T) {
	t.Parallel()

	called := false
	fn := withStartedCounter(func(ctx context.Context) error {
		called = true
		return nil
	}, metricnoop.Meter{}, "test")

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithStartedCounter_errorPassthrough(t *testing.T) {
	t.Parallel()

	fn := withStartedCounter(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, metricnoop.Meter{}, "test")

	require.EqualError(t, fn(context.Background()), "fail")
}

// ------------------------------------------------------------------------------------------------
// ~ withFinishedCounter
// ------------------------------------------------------------------------------------------------

func TestWithFinishedCounter(t *testing.T) {
	t.Parallel()

	called := false
	fn := withFinishedCounter(func(ctx context.Context) error {
		called = true
		return nil
	}, metricnoop.Meter{}, "test")

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithFinishedCounter_errorPassthrough(t *testing.T) {
	t.Parallel()

	fn := withFinishedCounter(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, metricnoop.Meter{}, "test")

	require.EqualError(t, fn(context.Background()), "fail")
}

// ------------------------------------------------------------------------------------------------
// ~ withErrorCounter
// ------------------------------------------------------------------------------------------------

func TestWithErrorCounter_noError(t *testing.T) {
	t.Parallel()

	called := false
	fn := withErrorCounter(func(ctx context.Context) error {
		called = true
		return nil
	}, metricnoop.Meter{}, "test")

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithErrorCounter_withError(t *testing.T) {
	t.Parallel()

	fn := withErrorCounter(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, metricnoop.Meter{}, "test")

	require.EqualError(t, fn(context.Background()), "fail")
}

// ------------------------------------------------------------------------------------------------
// ~ withActiveUpDownCounter
// ------------------------------------------------------------------------------------------------

func TestWithActiveUpDownCounter(t *testing.T) {
	t.Parallel()

	called := false
	fn := withActiveUpDownCounter(func(ctx context.Context) error {
		called = true
		return nil
	}, metricnoop.Meter{}, "test")

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithActiveUpDownCounter_errorPassthrough(t *testing.T) {
	t.Parallel()

	fn := withActiveUpDownCounter(func(ctx context.Context) error {
		return fmt.Errorf("fail")
	}, metricnoop.Meter{}, "test")

	require.EqualError(t, fn(context.Background()), "fail")
}

// ------------------------------------------------------------------------------------------------
// ~ withDurationHistogram
// ------------------------------------------------------------------------------------------------

func TestWithDurationHistogram(t *testing.T) {
	t.Parallel()

	called := false
	fn := withDurationHistogram(func(ctx context.Context) error {
		time.Sleep(time.Millisecond)
		called = true
		return nil
	}, metricnoop.Meter{}, "test")

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithDurationHistogram_errorPassthrough(t *testing.T) {
	t.Parallel()

	fn := withDurationHistogram(func(ctx context.Context) error {
		time.Sleep(time.Millisecond)
		return fmt.Errorf("fail")
	}, metricnoop.Meter{}, "test")

	require.EqualError(t, fn(context.Background()), "fail")
}

// ------------------------------------------------------------------------------------------------
// ~ withTracing
// ------------------------------------------------------------------------------------------------

func TestWithTracing_noError(t *testing.T) {
	t.Parallel()

	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	o := GoOptions{
		baseOptions: baseOptions{
			name:           "test-routine",
			tracerProvider: tp,
		},
	}

	called := false
	fn := withTracing(func(ctx context.Context) error {
		called = true
		return nil
	}, &o, 1)

	require.NoError(t, fn(context.Background()))
	assert.True(t, called)
}

func TestWithTracing_withError(t *testing.T) {
	t.Parallel()

	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t))

	o := GoOptions{
		baseOptions: baseOptions{
			name:           "test-routine",
			tracerProvider: tp,
		},
	}

	fn := withTracing(func(ctx context.Context) error {
		return fmt.Errorf("traced error")
	}, &o, 1)

	require.EqualError(t, fn(context.Background()), "traced error")
}

// ------------------------------------------------------------------------------------------------
// ~ WithMiddleware (via Go integration)
// ------------------------------------------------------------------------------------------------

func TestWithMiddleware_single(t *testing.T) {
	t.Parallel()

	var order []string

	m := Middleware(func(fn Func) Func {
		return func(ctx context.Context) error {
			order = append(order, "before")
			err := fn(ctx)
			order = append(order, "after")
			return err
		}
	})

	wrapped := m(func(ctx context.Context) error {
		order = append(order, "fn")
		return nil
	})

	require.NoError(t, wrapped(context.Background()))
	assert.Equal(t, []string{"before", "fn", "after"}, order)
}

func TestWithMiddleware_multiple(t *testing.T) {
	t.Parallel()

	var order []string

	m1 := Middleware(func(fn Func) Func {
		return func(ctx context.Context) error {
			order = append(order, "m1")
			return fn(ctx)
		}
	})

	m2 := Middleware(func(fn Func) Func {
		return func(ctx context.Context) error {
			order = append(order, "m2")
			return fn(ctx)
		}
	})

	// apply in order: m1 wraps first, m2 wraps second → m2 runs first (outermost)
	run := Func(func(ctx context.Context) error {
		order = append(order, "fn")
		return nil
	})

	run = m1(run)
	run = m2(run)

	require.NoError(t, run(context.Background()))
	assert.Equal(t, []string{"m2", "m1", "fn"}, order)
}

func TestWithMiddleware_errorPassthrough(t *testing.T) {
	t.Parallel()

	m := Middleware(func(fn Func) Func {
		return func(ctx context.Context) error {
			return fn(ctx)
		}
	})

	wrapped := m(func(ctx context.Context) error {
		return fmt.Errorf("inner error")
	})

	require.EqualError(t, wrapped(context.Background()), "inner error")
}

// ------------------------------------------------------------------------------------------------
// ~ handleError
// ------------------------------------------------------------------------------------------------

func TestHandleError_customHandler(t *testing.T) {
	t.Parallel()

	var got error
	handler := ErrorHandler(func(ctx context.Context, err error) {
		got = err
	})

	handleError(context.Background(), fmt.Errorf("test"), handler, nil, "test")
	require.EqualError(t, got, "test")
}

func TestHandleError_withLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, nil))

	handleError(context.Background(), fmt.Errorf("logged"), nil, l, "test")
	assert.Contains(t, buf.String(), "logged")
	assert.Contains(t, buf.String(), "gofuncy.go error")
}

func TestHandleError_defaultLogger(t *testing.T) {
	t.Parallel()

	// should not panic with nil handler and nil logger
	handleError(context.Background(), fmt.Errorf("default"), nil, nil, "test")
}
