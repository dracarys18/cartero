package utils

func FilterArray[T any](input []T, predicate func(T) bool) []T {
	filtered := make([]T, 0)
	for _, item := range input {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

