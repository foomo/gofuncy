package gofuncy_test

import (
	"context"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	"github.com/foomo/gofuncy/semconv"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const spawnEventName = "gofuncy.spawn"

// ------------------------------------------------------------------------------------------------
// ~ Go: spawn event on the caller's span by default
// ------------------------------------------------------------------------------------------------

func TestSpawnEvent_GoRecordsOnParent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	done := make(chan struct{})

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("worker"),
		gofuncy.WithTracerProvider(tp),
	)

	waitClosed(t, done)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	event, ok := findEvent(parent, spawnEventName)
	require.True(t, ok, "parent span should carry a %q event", spawnEventName)

	name, ok := attrValue(event.Attributes, semconv.RoutineNameKey)
	require.True(t, ok, "spawn event should carry the routine name")
	assert.Equal(t, "worker", name.AsString())
}

func TestSpawnEvent_GoNoParent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	done := make(chan struct{})

	// No parent span in context: nothing to record the event on, must not panic.
	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("orphan"),
		gofuncy.WithTracerProvider(tp),
	)

	waitClosed(t, done)

	// The detached go span still exports; ensure no span carries a spawn event.
	_ = findSpanEventually(t, exp, tp, "gofuncy.go")

	for _, s := range exp.GetSpans() {
		_, ok := findEvent(s, spawnEventName)
		assert.False(t, ok, "no span should carry a spawn event when there is no parent")
	}
}

func TestSpawnEvent_WithoutSpawnEvent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	done := make(chan struct{})

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("worker"),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithoutSpawnEvent(),
	)

	waitClosed(t, done)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	_, ok := findEvent(parent, spawnEventName)
	assert.False(t, ok, "WithoutSpawnEvent should suppress the spawn event")
}

func TestSpawnEvent_ChildTraceNoEvent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	done := make(chan struct{})

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("worker"),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithChildTrace(),
	)

	waitClosed(t, done)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	// A child routine already appears nested under the caller, so no breadcrumb.
	_, ok := findEvent(parent, spawnEventName)
	assert.False(t, ok, "child traces should not record a spawn event")
}

// ------------------------------------------------------------------------------------------------
// ~ Group: spawn event when detached
// ------------------------------------------------------------------------------------------------

func TestSpawnEvent_GroupDetached(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	g := gofuncy.NewGroup(ctx,
		gofuncy.WithName("detached-group"),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithDetachedTrace(),
	)

	g.Add(func(ctx context.Context) error { return nil })

	require.NoError(t, g.Wait())

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	event, ok := findEvent(parent, spawnEventName)
	require.True(t, ok, "detached group should record a spawn event on the parent")

	name, ok := attrValue(event.Attributes, semconv.RoutineNameKey)
	require.True(t, ok)
	assert.Equal(t, "detached-group", name.AsString())
}

func TestSpawnEvent_GroupChildDefaultNoEvent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	g := gofuncy.NewGroup(ctx,
		gofuncy.WithName("child-group"),
		gofuncy.WithTracerProvider(tp),
	)

	g.Add(func(ctx context.Context) error { return nil })

	require.NoError(t, g.Wait())

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	_, ok := findEvent(parent, spawnEventName)
	assert.False(t, ok, "child-default group should not record a spawn event")
}

// ------------------------------------------------------------------------------------------------
// ~ Code location: locks the caller-skip offset
// ------------------------------------------------------------------------------------------------

func TestSpawnEvent_recordsCodeLocation(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	done := make(chan struct{})

	gofuncy.Go(ctx,
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("worker"),
		gofuncy.WithTracerProvider(tp),
	)

	waitClosed(t, done)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	parent := findSpan(t, exp.GetSpans(), "parent")

	event, ok := findEvent(parent, spawnEventName)
	require.True(t, ok)

	fn, ok := attrValue(event.Attributes, otelsemconv.CodeFunctionNameKey)
	require.True(t, ok, "spawn event should carry the caller code location")
	assert.Contains(t, fn.AsString(), "TestSpawnEvent_recordsCodeLocation",
		"code.function.name should point at the call site, not gofuncy internals")
}

// ------------------------------------------------------------------------------------------------
// ~ Helpers
// ------------------------------------------------------------------------------------------------

func waitClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func findEvent(span tracetest.SpanStub, name string) (sdktrace.Event, bool) {
	for _, e := range span.Events {
		if e.Name == name {
			return e, true
		}
	}

	return sdktrace.Event{}, false
}

func attrValue(attrs []attribute.KeyValue, key attribute.Key) (attribute.Value, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value, true
		}
	}

	return attribute.Value{}, false
}
