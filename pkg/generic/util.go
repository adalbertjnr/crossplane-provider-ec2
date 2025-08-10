package generic

func FromSliceToMap[T any](collection []T, keyExtractorFunc func(T) string) map[string]struct{} {
	m := make(map[string]struct{}, len(collection))

	for _, item := range collection {
		k := keyExtractorFunc(item)
		m[k] = struct{}{}
	}

	return m
}

func FromSliceToMapWithValues[T any](collection []T, kvExtractorFunc func(T) (string, string)) map[string]string {
	m := make(map[string]string, len(collection))

	for i := range collection {
		key, value := kvExtractorFunc(collection[i])
		m[key] = value
	}

	return m
}
