package gofuncy

import (
	"context"
)

type contextKey int

type Context struct {
	context.Context //nolint:containedctx
}

const (
	// NameRoot is the default root name used for setting or retrieving the root context in a routine hierarchy.
	NameRoot string = "root"
	// NameNoName is the default name used when no routine name is explicitly set.
	NameNoName string = "noname"
)

const (
	contextKeyName contextKey = iota
	contextKeyParent
	contextKeyRoutine
)

// routineInfo stores both name and parent in a single context value to reduce allocations.
type routineInfo struct {
	name   string
	parent string
}

// Ctx helper
func Ctx(ctx context.Context) Context {
	return Context{Context: ctx}
}

// Name returns the routine name from the given context
func (c Context) Name() string {
	return NameFromContext(c)
}

// Parent returns the parent routine name from the given context
func (c Context) Parent() string {
	return ParentFromContext(c)
}

// Root returns the context with the `root` name set
func (c Context) Root() context.Context {
	return injectNameIntoContext(c.Context, NameRoot)
}

// NameFromContext extracts the routine name from the given context, falling back to a default "noname" if not found.
func NameFromContext(ctx context.Context) string {
	// check combined key first
	if ri, ok := ctx.Value(contextKeyRoutine).(routineInfo); ok {
		return ri.name
	}

	if value, ok := ctx.Value(contextKeyName).(string); ok {
		return value
	}

	return NameNoName
}

func injectParentIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeyParent, name)
}

func ParentFromContext(ctx context.Context) string {
	// check combined key first
	if ri, ok := ctx.Value(contextKeyRoutine).(routineInfo); ok {
		return ri.parent
	}

	if value, ok := ctx.Value(contextKeyParent).(string); ok {
		return value
	}

	return ""
}

func injectRoutineIntoContext(ctx context.Context, name, parent string) context.Context {
	return context.WithValue(ctx, contextKeyRoutine, routineInfo{name: name, parent: parent})
}

func injectNameIntoContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, contextKeyName, name)
}
