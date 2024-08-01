package gofuncy

import (
	"context"
)

type contextKey string

const (
	contextKeyRoutine       contextKey = "routine"
	contextKeyParentRoutine contextKey = "parentRoutine"
	contextKeySender        contextKey = "sender"
)

func injectRoutineIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeyRoutine, name)
}

func RoutineFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(contextKeyRoutine).(string); ok {
		return value
	}
	return "noname"
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
