package semconv

import "go.opentelemetry.io/otel/attribute"

// Attribute keys for gofuncy telemetry.
const (
	// RoutineNameKey is the attribute key for the goroutine name.
	RoutineNameKey = attribute.Key("gofuncy.routine.name")
	// RoutineParentKey is the attribute key for the parent goroutine name.
	RoutineParentKey = attribute.Key("gofuncy.routine.parent")
	// ChanNameKey is the attribute key for the channel name.
	ChanNameKey = attribute.Key("gofuncy.chan.name")
	// ChanCapKey is the attribute key for the channel buffer capacity.
	ChanCapKey = attribute.Key("gofuncy.chan.cap")
	// ChanSizeKey is the attribute key for the current channel buffer length.
	ChanSizeKey = attribute.Key("gofuncy.chan.size")
	// GroupSizeKey is the attribute key for the number of functions in a group.
	GroupSizeKey = attribute.Key("gofuncy.group.size")
	// ErrorKey is the attribute key indicating whether an error occurred.
	ErrorKey = attribute.Key("error")
)

// RoutineName returns an attribute with the goroutine name.
func RoutineName(v string) attribute.KeyValue {
	return RoutineNameKey.String(v)
}

// RoutineParent returns an attribute with the parent goroutine name.
func RoutineParent(v string) attribute.KeyValue {
	return RoutineParentKey.String(v)
}

// ChanName returns an attribute with the channel name.
func ChanName(v string) attribute.KeyValue {
	return ChanNameKey.String(v)
}

// ChanCap returns an attribute with the channel buffer capacity.
func ChanCap(v int) attribute.KeyValue {
	return ChanCapKey.Int(v)
}

// ChanSize returns an attribute with the current channel buffer length.
func ChanSize(v int) attribute.KeyValue {
	return ChanSizeKey.Int(v)
}

// GroupSize returns an attribute with the number of functions in a group.
func GroupSize(v int) attribute.KeyValue {
	return GroupSizeKey.Int(v)
}

// Error returns an attribute indicating whether an error occurred.
func Error(v bool) attribute.KeyValue {
	return ErrorKey.Bool(v)
}
