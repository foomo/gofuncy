package gofuncy_test

import (
	"testing"

	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicError_Error(t *testing.T) {
	t.Parallel()

	err := &gofuncy.PanicError{Value: "something went wrong"}
	assert.Equal(t, "panic: something went wrong", err.Error())
}

func TestPanicError_ErrorAs(t *testing.T) {
	t.Parallel()

	original := &gofuncy.PanicError{Value: 42, Stack: []byte("stack")}

	var wrapped error = original

	var target *gofuncy.PanicError

	require.ErrorAs(t, wrapped, &target)
	assert.Equal(t, 42, target.Value)
	assert.Equal(t, []byte("stack"), target.Stack)
}
