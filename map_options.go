package gofuncy

import "log/slog"

// MapOptions holds configuration for Map and MapBackground.
type MapOptions struct {
	concurrentOptions
}

// MapOptionsBuilder collects configuration for MapOptions.
type MapOptionsBuilder struct {
	Opts []func(*MapOptions) error
}

// MapOption creates a new MapOptionsBuilder.
func MapOption() *MapOptionsBuilder {
	return &MapOptionsBuilder{}
}

// List returns the list of MapOptions setter functions.
func (b *MapOptionsBuilder) List() []func(*MapOptions) error {
	return b.Opts
}

func newMapOptions(builders []*MapOptionsBuilder) *MapOptions {
	o := &MapOptions{}
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

func (b *MapOptionsBuilder) WithName(name string) *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.name = name
		return nil
	})

	return b
}

func (b *MapOptionsBuilder) WithLogger(l *slog.Logger) *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.l = l
		return nil
	})

	return b
}

func (b *MapOptionsBuilder) WithTracing() *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.tracing = true
		return nil
	})

	return b
}

func (b *MapOptionsBuilder) WithUpDownMetric() *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.upDownMetric = true
		return nil
	})

	return b
}

func (b *MapOptionsBuilder) WithDurationMetric() *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.durationMetric = true
		return nil
	})

	return b
}

func (b *MapOptionsBuilder) WithCounterMetric() *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.counterMetric = true
		return nil
	})

	return b
}

// WithLimit sets the maximum number of concurrent goroutines.
func (b *MapOptionsBuilder) WithLimit(n int) *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.limit = n
		return nil
	})

	return b
}

// WithFailFast cancels remaining goroutines on first error.
func (b *MapOptionsBuilder) WithFailFast() *MapOptionsBuilder {
	b.Opts = append(b.Opts, func(o *MapOptions) error {
		o.failFast = true
		return nil
	})

	return b
}
