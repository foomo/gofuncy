package gofuncy

// GroupOptions holds configuration for Group and GroupBackground.
type GroupOptions struct {
	concurrentOptions
}

func newGroupOptions(opts []Option[GroupOptions]) *GroupOptions {
	o := &GroupOptions{}
	o.name = NameNoName

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}
