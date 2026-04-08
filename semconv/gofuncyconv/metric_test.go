package gofuncyconv_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

func TestGoroutinesStarted(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesStarted(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.started", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Total number of goroutines started", m.Description())
	assert.NotNil(t, m.Inst())

	// must not panic
	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesStarted_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesStarted(nil)
	require.NoError(t, err)

	assert.Nil(t, m.Inst())

	// must not panic with nil instrument
	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesErrors(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesErrors(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.errors", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Total number of goroutine errors", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesErrors_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesErrors(nil)
	require.NoError(t, err)

	assert.Nil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesStalled(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesStalled(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.stalled", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Total number of goroutines that exceeded their stall threshold", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesStalled_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesStalled(nil)
	require.NoError(t, err)

	assert.Nil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesActive(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesActive(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.active", m.Name())
	assert.Equal(t, "{goroutine}", m.Unit())
	assert.Equal(t, "Number of currently active goroutines", m.Description())
	assert.NotNil(t, m.Inst())

	m.Add(context.Background(), 1, "test-routine")
	m.Add(context.Background(), -1, "test-routine")
}

func TestGoroutinesActive_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesActive(nil)
	require.NoError(t, err)

	m.Add(context.Background(), 1, "test-routine")
}

func TestGoroutinesDuration(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGoroutinesDuration(noop.Meter{})
	require.NoError(t, err)

	assert.Equal(t, "gofuncy.goroutines.duration.seconds", m.Name())
	assert.Equal(t, "s", m.Unit())
	assert.Equal(t, "Duration of goroutine execution", m.Description())
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

	m.Record(context.Background(), 0.5, "test-group", false)
	m.Record(context.Background(), 1.0, "test-group", true)
}

func TestGroupsDuration_nilMeter(t *testing.T) {
	t.Parallel()

	m, err := gofuncyconv.NewGroupsDuration(nil)
	require.NoError(t, err)

	m.Record(context.Background(), 0.5, "test-group", false)
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
