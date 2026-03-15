package gofuncyconv_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

func TestGoroutinesTotal(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesTotal(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.total", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Gofuncy running go routine count", m.Description())
	assert.NotNil(t, m.Inst())

	// must not panic
	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesTotal_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesTotal(nil)
	require.NoError(t, err)

	assert.Nil(t, m.Inst())

	// must not panic with nil instrument
	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesCurrent(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesCurrent(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.current", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Gofuncy running go routine up/down count", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
	m.Add(context.Background(), -1, "test-routine")
}

func TestGoroutinesCurrent_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesCurrent(nil)
	require.NoError(t, err)

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesDuration(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesDuration(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.duration.seconds", m.Name())
	assert.Equal(t, "s", m.Unit())
	assert.Equal(t, "Gofuncy go routine duration histogram", m.Description())
	assert.NotNil(t, m.Inst())

	m.Record(context.Background(), 0.5, "test-routine", false)
	m.Record(context.Background(), 1.0, "test-routine", true)
}

func TestGoroutinesDuration_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesDuration(nil)
	require.NoError(t, err)

	m.Record(context.Background(), 0.5, "test-routine", false)
}

func TestGroupsDuration(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGroupsDuration(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.groups.duration.seconds", m.Name())
	assert.Equal(t, "s", m.Unit())
	assert.Equal(t, "Gofuncy group/map duration histogram", m.Description())
	assert.NotNil(t, m.Inst())

	m.Record(context.Background(), 0.5, "test-group", false, 3)
	m.Record(context.Background(), 1.0, "test-group", true, 5)
}

func TestGroupsDuration_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGroupsDuration(nil)
	require.NoError(t, err)

	m.Record(context.Background(), 0.5, "test-group", false, 3)
}

func TestChansCurrent(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewChansCurrent(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.chans.current", m.Name())
	assert.Equal(t, "{chan}", m.Unit())
	assert.Equal(t, "Gofuncy open chan up/down count", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-chan")
	m.Add(context.Background(), -1, "test-chan")
}

func TestChansCurrent_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewChansCurrent(nil)
	require.NoError(t, err)

	m.Add(context.Background(), 1, "test-chan")
}

func TestMessagesCurrent(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewMessagesCurrent(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.messages.current", m.Name())
	assert.Equal(t, "{message}", m.Unit())
	assert.Equal(t, "Gofuncy pending message count", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-chan")
	m.Add(context.Background(), -1, "test-chan")
}

func TestMessagesCurrent_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewMessagesCurrent(nil)
	require.NoError(t, err)

	m.Add(context.Background(), 1, "test-chan")
}

func TestMessagesDuration(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewMessagesDuration(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.messages.duration.seconds", m.Name())
	assert.Equal(t, "s", m.Unit())
	assert.Equal(t, "Gofuncy chan message send duration", m.Description())
	assert.NotNil(t, m.Inst())

	m.Record(context.Background(), 0.5, "test-chan")
}

func TestMessagesDuration_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewMessagesDuration(nil)
	require.NoError(t, err)

	m.Record(context.Background(), 0.5, "test-chan")
}
