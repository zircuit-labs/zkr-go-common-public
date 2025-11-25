// Package collections provides generic data structures.
package collections

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"

	zkriter "github.com/zircuit-labs/zkr-go-common/iter"
)

// Set represents a mathematical set of comparable elements.
// It is implemented as a map with empty struct values for memory efficiency.
type Set[T comparable] map[T]struct{}

// NewSet creates a new set containing the given values.
func NewSet[T comparable](vals ...T) Set[T] {
	s := make(Set[T], len(vals))
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// Add adds the given values to the set.
func (s Set[T]) Add(vals ...T) {
	s.AddIter(slices.Values(vals))
}

// AddIter adds all values from the iterator to the set.
func (s Set[T]) AddIter(vals iter.Seq[T]) {
	for v := range vals {
		s[v] = struct{}{}
	}
}

// Remove removes the given values from the set.
func (s Set[T]) Remove(vals ...T) {
	s.RemoveIter(slices.Values(vals))
}

// RemoveIter removes all values from the iterator from the set.
func (s Set[T]) RemoveIter(vals iter.Seq[T]) {
	for v := range vals {
		delete(s, v)
	}
}

// Iter returns an iterator over the elements in the set.
func (s Set[T]) Iter() iter.Seq[T] {
	return maps.Keys(s)
}

// Members returns all elements in the set as a slice.
func (s Set[T]) Members() []T {
	return slices.Collect(s.Iter())
}

// String returns a string representation of the set.
func (s Set[T]) String() string {
	return fmt.Sprintf("%v", s.Members())
}

// Contains returns true if the set contains all of the given values.
func (s Set[T]) Contains(vals ...T) bool {
	return zkriter.And(func(v T) bool {
		_, ok := s[v]
		return ok
	}, slices.Values(vals))
}

// ContainsAny returns true if the set contains at least one of the given values.
func (s Set[T]) ContainsAny(vals ...T) bool {
	return zkriter.Or(func(v T) bool {
		_, ok := s[v]
		return ok
	}, slices.Values(vals))
}

// Size returns the number of elements in the set.
func (s Set[T]) Size() int {
	return len(s)
}

// Empty returns true if the set contains no elements.
func (s Set[T]) Empty() bool {
	return len(s) == 0
}

// Equal returns true if s and s2 are identical sets.
func (s Set[T]) Equal(s2 Set[T]) bool {
	if len(s) != len(s2) {
		return false
	}
	return zkriter.And(func(v T) bool {
		_, ok := s2[v]
		return ok
	}, s.Iter())
}

// Clear removes all elements from s.
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Clone returns a copy of s.
func (s Set[T]) Clone() Set[T] {
	return maps.Clone(s)
}

// Union returns a new set containing all elements from both sets.
func (s Set[T]) Union(s2 Set[T]) Set[T] {
	result := maps.Clone(s)
	result.AddIter(s2.Iter())
	return result
}

// Intersection returns a new set containing only elements present in both sets.
func (s Set[T]) Intersection(s2 Set[T]) Set[T] {
	result := NewSet[T]()
	result.AddIter(zkriter.Filter(func(v T) bool { return s2.Contains(v) }, s.Iter()))
	return result
}

// Difference returns a new set containing elements in s but not in s2.
func (s Set[T]) Difference(s2 Set[T]) Set[T] {
	result := NewSet[T]()
	result.AddIter(zkriter.Filter(func(v T) bool { return !s2.Contains(v) }, s.Iter()))
	return result
}

// SymmetricDifference returns a new set containing elements that are in s or s2 but not both.
func (s Set[T]) SymmetricDifference(s2 Set[T]) Set[T] {
	return s.Difference(s2).Union(s2.Difference(s))
}

// Filter returns a new set containing only elements that satisfy the predicate.
func (s Set[T]) Filter(p zkriter.Predicate[T]) Set[T] {
	result := NewSet[T]()
	result.AddIter(zkriter.Filter(p, s.Iter()))
	return result
}

// TransformSet applies a transformation function to each element in the set
// and returns a new set containing the transformed elements.
func TransformSet[S, T comparable](s Set[S], transform zkriter.Transformation[S, T]) Set[T] {
	result := NewSet[T]()
	result.AddIter(zkriter.Transform(transform, s.Iter()))
	return result
}

// MarshalJSON implements json.Marshaler interface.
// The set is marshaled as a JSON array containing all elements.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	members := s.Members()
	if members == nil {
		members = []T{}
	}
	return json.Marshal(members)
}

// UnmarshalJSON implements json.Unmarshaler interface.
// The set is unmarshaled from a JSON array of elements.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	var members []T
	if err := json.Unmarshal(data, &members); err != nil {
		return err
	}

	*s = NewSet(members...)
	return nil
}
