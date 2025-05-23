package gofuncy

import (
	"context"
)

type contextKey string

const (
	NoNameRoutine           string     = "noname"
	contextKeyRoutine       contextKey = "routine"
	contextKeyParentRoutine contextKey = "parentRoutine"
	contextKeySender        contextKey = "sender"
)

func RootContext(ctx context.Context) context.Context {
	return injectRoutineIntoContext(ctx, "root")
}

func injectRoutineIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeyRoutine, name)
}

func RoutineFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(contextKeyRoutine).(string); ok {
		return value
	}
	return NoNameRoutine
}

func injectSenderIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeySender, name)
}

func SenderFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(contextKeySender).(string); ok {
		return value
	}
	return ""
}

func injectParentRoutineIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeyParentRoutine, name)
}

func ParentRoutineFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(contextKeyParentRoutine).(string); ok {
		return value
	}
	return ""
}
