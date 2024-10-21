package fn

func Map[S []InType, U []OutType, InType any, OutType any](in S, fn func(InType) OutType) U {
	out := make([]OutType, len(in))
	for i := range in {
		out[i] = fn(in[i])
	}
	return out
}
