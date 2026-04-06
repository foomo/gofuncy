package gofuncy

// GoOptions holds configuration for Go and GoBackground.
type GoOptions struct {
	baseOptions
}

func newGoOptions(opts []Option[GoOptions]) *GoOptions {
	o := &GoOptions{
		baseOptions: baseOptions{
			name: NameNoName,
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}
