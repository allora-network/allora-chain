package fn

func Map[S []InType, U []OutType, InType any, OutType any](in S, fn func(InType) OutType) U {
	out := make([]OutType, len(in))
	for i := range in {
		out[i] = fn(in[i])
	}
	return out
}

func ElementsMatch[S []InType, InType comparable](a, b S) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[InType]struct{}, len(a))
	for i := range a {
		seen[a[i]] = struct{}{}
	}
	for i := range b {
		if _, ok := seen[b[i]]; !ok {
			return false
		}
	}
	return true
}
