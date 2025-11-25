package iter

import "iter"

type Predicate[V any] func(V) bool

// Filter returns a sequence that contains the elements of s for which p returns true.
func Filter[V any](p Predicate[V], s iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range s {
			if p(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}
