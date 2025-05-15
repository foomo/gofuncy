package gofuncy_test

import (
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
)

func TestCtx_NoName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, gofuncy.NameNoName, gofuncy.Ctx(t.Context()).Name())
}

func TestCtx_Root(t *testing.T) {
	t.Parallel()
	ctx := gofuncy.Ctx(t.Context()).Root()
	assert.Equal(t, gofuncy.NameRoot, gofuncy.Ctx(ctx).Name())
}

func TestCtx_Parent(t *testing.T) {
	t.Parallel()
	assert.Empty(t, gofuncy.Ctx(t.Context()).Parent())
}
