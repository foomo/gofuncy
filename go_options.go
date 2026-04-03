package gofuncy

import (
	"log/slog"
	"time"

	optionx "github.com/foomo/go/option"
)

// GoOptions holds configuration for Go and GoBackground.
type GoOptions struct {
	baseOptions
	// timeout      time.Duration
	// callerSkip   int
	// errorHandler ErrorHandler
}

// OptionsBuilder collects configuration for GoOptions.
type OptionsBuilder struct {
	optionx.Builder[*GoOptions]
}

// GoOption creates a new OptionsBuilder.
func GoOption() *OptionsBuilder {
	return &OptionsBuilder{
		optionx.Builder[*GoOptions]{},
	}
}

func (b *OptionsBuilder) WithName(name string) *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.name = name
	})

	return b
}

func (b *OptionsBuilder) WithTimeout(timeout time.Duration) *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.timeout = timeout
	})

	return b
}

func (b *OptionsBuilder) WithCallerSkip(skip int) *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.callerSkip = skip
	})

	return b
}

func (b *OptionsBuilder) WithLogger(l *slog.Logger) *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.l = l
	})

	return b
}

func (b *OptionsBuilder) WithTracing() *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.tracing = true
	})

	return b
}

func (b *OptionsBuilder) WithUpDownMetric() *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.upDownMetric = true
	})

	return b
}

func (b *OptionsBuilder) WithDurationMetric() *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.durationMetric = true
	})

	return b
}

func (b *OptionsBuilder) WithCounterMetric() *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.counterMetric = true
	})

	return b
}

func (b *OptionsBuilder) WithErrorHandler(h ErrorHandler) *OptionsBuilder {
	b.Opts = append(b.Opts, func(o *GoOptions) {
		o.errorHandler = h
	})

	return b
}

func newOptions(builders []*OptionsBuilder) *GoOptions {
	o := &GoOptions{
		baseOptions: baseOptions{
			name: NameNoName,
		},
	}

	for _, b := range builders {
		if b != nil {
			optionx.Apply(o, b.List()...)
		}
	}

	return o
}
