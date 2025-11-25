package iter

import "iter"

type Transformation[S, T any] func(S) T

// Transform applies t to each elements of s, returning the sequence of the transformed elements
func Transform[S, T any](t Transformation[S, T], s iter.Seq[S]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s {
			if !yield(t(v)) {
				return
			}
		}
	}
}
