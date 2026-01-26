package internal

type From[T any] interface {
	From(T)
}

type Into[T any] interface {
	Into() T
}

type TryFrom[T any] interface {
	TryFrom(T) error
}

type TryInto[T any] interface {
	TryInto() (T, error)
}
