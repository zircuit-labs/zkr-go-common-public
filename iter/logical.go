package iter

import "iter"

// And returns true if the predicate returns true for all elements of s.
func And[V any](p Predicate[V], s iter.Seq[V]) bool {
	for v := range s {
		if !p(v) {
			return false
		}
	}
	return true
}

// Or returns true if the predicate returns true for any element of s.
func Or[V any](p Predicate[V], s iter.Seq[V]) bool {
	for v := range s {
		if p(v) {
			return true
		}
	}
	return false
}
