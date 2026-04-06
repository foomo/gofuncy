package gofuncy

// MapOptions holds configuration for Map and MapBackground.
type MapOptions struct {
	concurrentOptions
}

func newMapOptions(opts []Option[MapOptions]) *MapOptions {
	o := &MapOptions{}
	o.name = NameNoName

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}
