package gofuncy

type Message[T any] struct {
	sender string
	value  T
}

func (m Message[T]) Sender() string {
	return m.sender
}

func (m Message[T]) Value() T {
	return m.value
}
