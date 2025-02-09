package utils

func Distinct[T comparable](sites []T) []T {
	m := make(map[T]struct{})
	for _, s := range sites {
		m[s] = struct{}{}
	}
	u := make([]T, 0, len(m))
	for k := range m {
		u = append(u, k)
	}
	return u
}
