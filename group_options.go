package gofuncy

import "log/slog"

// GroupOptions holds configuration for Group and GroupBackground.
type GroupOptions struct {
	concurrentOptions
}

// GroupOptionsBuilder collects configuration for GroupOptions.
type GroupOptionsBuilder struct {
	Opts []func(*GroupOptions) error
}

// GroupOption creates a new GroupOptionsBuilder.
func GroupOption() *GroupOptionsBuilder {
	return &GroupOptionsBuilder{}
}

// List returns the list of GroupOptions setter functions.
func (b *GroupOptionsBuilder) List() []func(*GroupOptions) error {
	return b.Opts
}

func newGroupOptions(builders []*GroupOptionsBuilder) *GroupOptions {
	o := &GroupOptions{}
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

func (b *GroupOptionsBuilder) WithName(name string) *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.name = name
		return nil
	})

	return b
}

func (b *GroupOptionsBuilder) WithLogger(l *slog.Logger) *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.l = l
		return nil
	})

	return b
}

func (b *GroupOptionsBuilder) WithTracing() *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.tracing = true
		return nil
	})

	return b
}

func (b *GroupOptionsBuilder) WithUpDownMetric() *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.upDownMetric = true
		return nil
	})

	return b
}

func (b *GroupOptionsBuilder) WithDurationMetric() *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.durationMetric = true
		return nil
	})

	return b
}

func (b *GroupOptionsBuilder) WithCounterMetric() *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.counterMetric = true
		return nil
	})

	return b
}

// WithLimit sets the maximum number of concurrent goroutines.
func (b *GroupOptionsBuilder) WithLimit(n int) *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.limit = n
		return nil
	})

	return b
}

// WithFailFast cancels remaining goroutines on first error.
func (b *GroupOptionsBuilder) WithFailFast() *GroupOptionsBuilder {
	b.Opts = append(b.Opts, func(o *GroupOptions) error {
		o.failFast = true
		return nil
	})

	return b
}
