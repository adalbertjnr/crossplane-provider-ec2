package generic

func FromSliceToMap[T any](collection []T, keyExtractorFunc func(T) string) map[string]struct{} {
	m := make(map[string]struct{}, len(collection))

	for _, item := range collection {
		k := keyExtractorFunc(item)
		m[k] = struct{}{}
	}

	return m
}
