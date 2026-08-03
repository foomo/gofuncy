package gofuncy

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	goruntime "github.com/foomo/go/runtime"
	"github.com/foomo/gofuncy/semconv"
)

// spawnEventName is the low-cardinality name of the span event recorded on the
// caller's active span when a background routine is launched detached.
const spawnEventName = "gofuncy.spawn"

// recordSpawnEvent adds a "gofuncy.spawn" event to the caller's active span,
// marking that a background routine was launched. It must be called synchronously
// at the launch site (before the goroutine starts): the detached routine's own
// span is created inside the goroutine, by which point a fire-and-forget parent
// span may already have ended, and AddEvent on an ended span is a no-op.
//
// Only detached routines create a new root span, so only they need this
// breadcrumb; child routines already appear nested under the caller. The event is
// suppressed when tracing is off, when WithoutSpawnEvent was used, or when no
// valid parent span is present.
func recordSpawnEvent(ctx context.Context, o *options, callerSkip int) {
	if !o.tracing || !o.spawnEvent || !o.detachedTrace {
		return
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, 4)
	attrs = append(attrs, semconv.RoutineName(o.name))

	if fr := goruntime.CallFrame(callerSkip); !fr.Zero() {
		attrs = append(attrs,
			otelsemconv.CodeFilePath(fr.File),
			otelsemconv.CodeLineNumber(fr.Line),
			otelsemconv.CodeFunctionName(fr.Name()),
		)
	}

	span.AddEvent(spawnEventName, trace.WithAttributes(attrs...))
}
