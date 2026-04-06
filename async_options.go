package gofuncy

// AsyncOptions holds configuration for Async and AsyncBackground.
type AsyncOptions struct {
	baseOptions
}

func newAsyncOptions(opts []Option[AsyncOptions]) *AsyncOptions {
	o := &AsyncOptions{}
	o.name = NameNoName

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}
