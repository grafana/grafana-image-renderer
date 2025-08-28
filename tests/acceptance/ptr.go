package acceptance

func ptr[T any](v T) *T {
	return &v
}
