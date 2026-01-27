package ptr

func To[T any](val T) *T {
	return &val
}
