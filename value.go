package gofuncy

import (
	"context"
)

type Value[T any] struct {
	ctx  context.Context
	Data T
}

func (m *Value[T]) Context() context.Context {
	return m.ctx
}