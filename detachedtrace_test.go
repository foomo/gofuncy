package gofuncy_test

import (
	"context"
	"testing"
	"time"

	"github.com/foomo/gofuncy"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// ------------------------------------------------------------------------------------------------
// ~ Go: detached by default
// ------------------------------------------------------------------------------------------------

func TestGo_detachedTraceByDefault(t *testing.T) {
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
		gofuncy.WithName("detached-default"),
		gofuncy.WithTracerProvider(tp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	goSpan := findSpan(t, spans, "gofuncy.go detached-default")

	// Should be a root span (no parent)
	assert.False(t, goSpan.Parent.IsValid(), "Go span should be a root span (detached)")

	// Should have a link to the parent
	require.Len(t, goSpan.Links, 1, "Go span should have exactly one link")
	assert.Equal(t, parentSpan.SpanContext().TraceID(), goSpan.Links[0].SpanContext.TraceID())
	assert.Equal(t, parentSpan.SpanContext().SpanID(), goSpan.Links[0].SpanContext.SpanID())

	// Should have a different trace ID (new root)
	assert.NotEqual(t, parentSpan.SpanContext().TraceID(), goSpan.SpanContext.TraceID(),
		"detached span should have its own trace ID")
}

func TestGo_withChildTrace(t *testing.T) {
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
		gofuncy.WithName("child-override"),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithChildTrace(),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	goSpan := findSpan(t, spans, "gofuncy.go child-override")

	// Should be a child span (has parent)
	assert.True(t, goSpan.Parent.IsValid(), "Go span should be a child span with WithChildTrace")
	assert.Equal(t, parentSpan.SpanContext().TraceID(), goSpan.SpanContext.TraceID(),
		"child span should share parent's trace ID")

	// Should have no links
	assert.Empty(t, goSpan.Links, "child span should have no links")
}

func TestGo_detachedTraceNoParent(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	done := make(chan struct{})

	// No parent span in context
	gofuncy.Go(t.Context(),
		func(ctx context.Context) error {
			close(done)
			return nil
		},
		gofuncy.WithName("no-parent"),
		gofuncy.WithTracerProvider(tp),
	)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	goSpan := findSpan(t, spans, "gofuncy.go no-parent")

	// Root span with no links (no parent to link to)
	assert.False(t, goSpan.Parent.IsValid())
	assert.Empty(t, goSpan.Links)
}

// ------------------------------------------------------------------------------------------------
// ~ Do: child by default, opt-in detached
// ------------------------------------------------------------------------------------------------

func TestDo_childTraceByDefault(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	err := gofuncy.Do(ctx,
		func(ctx context.Context) error { return nil },
		gofuncy.WithName("child-default"),
		gofuncy.WithTracerProvider(tp),
	)
	require.NoError(t, err)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	doSpan := findSpan(t, spans, "gofuncy.do child-default")

	// Should be a child span
	assert.True(t, doSpan.Parent.IsValid(), "Do span should be a child span by default")
	assert.Equal(t, parentSpan.SpanContext().TraceID(), doSpan.SpanContext.TraceID())
	assert.Empty(t, doSpan.Links)
}

func TestDo_withDetachedTrace(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	tp := oteltesting.ReportTraces(t, exp)

	tracer := tp.Tracer("test")
	ctx, parentSpan := tracer.Start(t.Context(), "parent")

	err := gofuncy.Do(ctx,
		func(ctx context.Context) error { return nil },
		gofuncy.WithName("detached-opt-in"),
		gofuncy.WithTracerProvider(tp),
		gofuncy.WithDetachedTrace(),
	)
	require.NoError(t, err)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	doSpan := findSpan(t, spans, "gofuncy.do detached-opt-in")

	// Should be a root span with link
	assert.False(t, doSpan.Parent.IsValid(), "Do span should be detached with WithDetachedTrace")
	require.Len(t, doSpan.Links, 1)
	assert.Equal(t, parentSpan.SpanContext().SpanID(), doSpan.Links[0].SpanContext.SpanID())
	assert.NotEqual(t, parentSpan.SpanContext().TraceID(), doSpan.SpanContext.TraceID())
}

// ------------------------------------------------------------------------------------------------
// ~ Group: child by default, opt-in detached
// ------------------------------------------------------------------------------------------------

func TestGroup_withDetachedTrace(t *testing.T) {
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

	g.Add(func(ctx context.Context) error { return nil }, gofuncy.WithName("task-a"))
	g.Add(func(ctx context.Context) error { return nil }, gofuncy.WithName("task-b"))

	err := g.Wait()
	require.NoError(t, err)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	groupSpan := findSpan(t, spans, "gofuncy.group detached-group")

	// Group parent span should be a root with link to original parent
	assert.False(t, groupSpan.Parent.IsValid(), "group span should be a root span")
	require.Len(t, groupSpan.Links, 1)
	assert.Equal(t, parentSpan.SpanContext().SpanID(), groupSpan.Links[0].SpanContext.SpanID())

	// Child Add spans should be children of the group root span (not detached)
	taskA := findSpan(t, spans, "gofuncy.group.add task-a")
	taskB := findSpan(t, spans, "gofuncy.group.add task-b")

	assert.True(t, taskA.Parent.IsValid(), "task-a should be a child of the group span")
	assert.Equal(t, groupSpan.SpanContext.SpanID(), taskA.Parent.SpanID())
	assert.Empty(t, taskA.Links, "task-a should have no links")

	assert.True(t, taskB.Parent.IsValid(), "task-b should be a child of the group span")
	assert.Equal(t, groupSpan.SpanContext.SpanID(), taskB.Parent.SpanID())
	assert.Empty(t, taskB.Links, "task-b should have no links")
}

func TestGroup_childTraceByDefault(t *testing.T) {
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

	err := g.Wait()
	require.NoError(t, err)

	parentSpan.End()
	tp.ForceFlush(t.Context())

	spans := exp.GetSpans()
	groupSpan := findSpan(t, spans, "gofuncy.group child-group")

	// Should be a child span by default
	assert.True(t, groupSpan.Parent.IsValid(), "group span should be a child span by default")
	assert.Equal(t, parentSpan.SpanContext().TraceID(), groupSpan.SpanContext.TraceID())
	assert.Empty(t, groupSpan.Links)
}

// ------------------------------------------------------------------------------------------------
// ~ Helpers
// ------------------------------------------------------------------------------------------------

func findSpan(t *testing.T, spans tracetest.SpanStubs, name string) tracetest.SpanStub {
	t.Helper()

	for _, s := range spans {
		if s.Name == name {
			return s
		}
	}

	var names []string
	for _, s := range spans {
		names = append(names, s.Name)
	}

	t.Fatalf("span %q not found in %v", name, names)

	return tracetest.SpanStub{}
}
