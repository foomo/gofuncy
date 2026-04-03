package gofuncy

import "log/slog"

// AsyncOptions holds configuration for Async and AsyncBackground.
type AsyncOptions struct {
	baseOptions
}

// AsyncOptionsBuilder collects configuration for AsyncOptions.
type AsyncOptionsBuilder struct {
	Opts []func(*AsyncOptions) error
}

// AsyncOption creates a new AsyncOptionsBuilder.
func AsyncOption() *AsyncOptionsBuilder {
	return &AsyncOptionsBuilder{}
}

// List returns the list of AsyncOptions setter functions.
func (b *AsyncOptionsBuilder) List() []func(*AsyncOptions) error {
	return b.Opts
}

func newAsyncOptions(builders []*AsyncOptionsBuilder) *AsyncOptions {
	o := &AsyncOptions{}
	o.name = NameNoName

	for _, b := range builders {
		if b == nil {
			continue
		}

		for _, opt := range b.Opts {
			if opt != nil {
				_ = opt(o)
			}
		}
	}

	return o
}

func (b *AsyncOptionsBuilder) WithName(name string) *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.name = name
		return nil
	})

	return b
}

func (b *AsyncOptionsBuilder) WithLogger(l *slog.Logger) *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.l = l
		return nil
	})

	return b
}

func (b *AsyncOptionsBuilder) WithTracing() *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.tracing = true
		return nil
	})

	return b
}

func (b *AsyncOptionsBuilder) WithUpDownMetric() *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.upDownMetric = true
		return nil
	})

	return b
}

func (b *AsyncOptionsBuilder) WithDurationMetric() *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.durationMetric = true
		return nil
	})

	return b
}

func (b *AsyncOptionsBuilder) WithCounterMetric() *AsyncOptionsBuilder {
	b.Opts = append(b.Opts, func(o *AsyncOptions) error {
		o.counterMetric = true
		return nil
	})

	return b
}
