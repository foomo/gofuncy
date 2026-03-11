package semconv

import "go.opentelemetry.io/otel/attribute"

const (
	RoutineName   = attribute.Key("gofuncy.routine.name")
	RoutineParent = attribute.Key("gofuncy.routine.parent")
	ChanName      = attribute.Key("gofuncy.chan.name")
	ChanCap       = attribute.Key("gofuncy.chan.cap")
	ChanSize      = attribute.Key("gofuncy.chan.size")
)
