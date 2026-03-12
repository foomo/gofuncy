package semconv

import "go.opentelemetry.io/otel/attribute"

const (
	RoutineNameKey   = attribute.Key("gofuncy.routine.name")
	RoutineParentKey = attribute.Key("gofuncy.routine.parent")
	ChanNameKey      = attribute.Key("gofuncy.chan.name")
	ChanCapKey       = attribute.Key("gofuncy.chan.cap")
	ChanSizeKey      = attribute.Key("gofuncy.chan.size")
)

func RoutineName(v string) attribute.KeyValue {
	return RoutineNameKey.String(v)
}

func RoutineParent(v string) attribute.KeyValue {
	return RoutineParentKey.String(v)
}

func ChanName(v string) attribute.KeyValue {
	return ChanNameKey.String(v)
}

func ChanCap(v int) attribute.KeyValue {
	return ChanCapKey.Int(v)
}

func ChanSize(v int) attribute.KeyValue {
	return ChanSizeKey.Int(v)
}
