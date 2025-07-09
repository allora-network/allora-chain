package fn

import "golang.org/x/exp/constraints"

func Map[S []InType, U []OutType, InType any, OutType any](in S, fn func(InType) OutType) U {
	out := make([]OutType, len(in))
	for i := range in {
		out[i] = fn(in[i])
	}
	return out
}

func Find[S []InType, InType any](in S, fn func(InType) bool) (int, bool) {
	for i, item := range in {
		if fn(item) {
			return i, true
		}
	}
	return 0, false
}

func FilterFn[S []InType, InType any](in S, fn func(InType) bool) S {
	out := make(S, 0, len(in))
	for _, item := range in {
		if fn(item) {
			out = append(out, item)
		}
	}
	return out
}

func Trim[S ~[]InType, InType any, I constraints.Integer](in S, n I) S {
	if n >= I(len(in)) {
		return nil
	}
	return in[:I(len(in))-n]
}
