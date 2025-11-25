package collections_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/collections"
)

func TestNewSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "empty set",
			input:    []int{},
			expected: nil,
		},
		{
			name:     "single element",
			input:    []int{1},
			expected: []int{1},
		},
		{
			name:     "multiple elements",
			input:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "duplicate elements",
			input:    []int{1, 2, 2, 3, 1},
			expected: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			set := collections.NewSet(tt.input...)
			result := set.Members()
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSetAdd(t *testing.T) {
	t.Parallel()

	set := collections.NewSet[int]()
	set.Add(1, 2, 3)

	assert.ElementsMatch(t, []int{1, 2, 3}, set.Members())
}

func TestSetAddIter(t *testing.T) {
	t.Parallel()

	set := collections.NewSet[int]()
	values := []int{1, 2, 3, 4, 5}
	set.AddIter(slices.Values(values))

	assert.ElementsMatch(t, values, set.Members())
}

func TestSetRemove(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3, 4, 5)
	set.Remove(2, 4)

	assert.ElementsMatch(t, []int{1, 3, 5}, set.Members())
}

func TestSetRemoveIter(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3, 4, 5)
	toRemove := []int{2, 4}
	set.RemoveIter(slices.Values(toRemove))

	assert.ElementsMatch(t, []int{1, 3, 5}, set.Members())
}

func TestSetContains(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3)

	// Test single elements
	assert.True(t, set.Contains(1))
	assert.True(t, set.Contains(2))
	assert.True(t, set.Contains(3))
	assert.False(t, set.Contains(4))

	// Test multiple elements
	assert.True(t, set.Contains(1, 2))
	assert.True(t, set.Contains(1, 2, 3))
	assert.False(t, set.Contains(1, 4))
	assert.False(t, set.Contains(4, 5))

	// Test empty input
	assert.True(t, set.Contains())
}

func TestSetContainsAny(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3)

	// Test single elements
	assert.True(t, set.ContainsAny(1))
	assert.True(t, set.ContainsAny(2))
	assert.True(t, set.ContainsAny(3))
	assert.False(t, set.ContainsAny(4))

	// Test multiple elements
	assert.True(t, set.ContainsAny(1, 4))
	assert.True(t, set.ContainsAny(4, 2))
	assert.True(t, set.ContainsAny(1, 2, 3))
	assert.False(t, set.ContainsAny(4, 5, 6))

	// Test empty input
	assert.False(t, set.ContainsAny())
}

func TestSetSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []int
		expected int
	}{
		{
			name:     "empty set",
			input:    []int{},
			expected: 0,
		},
		{
			name:     "single element",
			input:    []int{1},
			expected: 1,
		},
		{
			name:     "multiple elements",
			input:    []int{1, 2, 3},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			set := collections.NewSet(tt.input...)
			assert.Equal(t, tt.expected, set.Size())
		})
	}
}

func TestSetEqual(t *testing.T) {
	t.Parallel()

	set1 := collections.NewSet(1, 2, 3)
	set2 := collections.NewSet(3, 2, 1)    // Same elements, different order
	set3 := collections.NewSet(1, 2)       // Subset
	set4 := collections.NewSet(1, 2, 3, 4) // Superset
	empty1 := collections.NewSet[int]()
	empty2 := collections.NewSet[int]()

	// Equal sets
	assert.True(t, set1.Equal(set2))
	assert.True(t, set2.Equal(set1))
	assert.True(t, empty1.Equal(empty2))

	// Unequal sets
	assert.False(t, set1.Equal(set3))
	assert.False(t, set1.Equal(set4))
	assert.False(t, set1.Equal(empty1))
	assert.False(t, empty1.Equal(set1))
}

func TestSetEmpty(t *testing.T) {
	t.Parallel()

	emptySet := collections.NewSet[int]()
	nonEmptySet := collections.NewSet(1, 2, 3)

	assert.True(t, emptySet.Empty())
	assert.False(t, nonEmptySet.Empty())

	// After clearing
	nonEmptySet.Clear()
	assert.True(t, nonEmptySet.Empty())
}

func TestSetClear(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3, 4, 5)
	assert.Equal(t, 5, set.Size())
	assert.False(t, set.Empty())

	set.Clear()

	assert.Equal(t, 0, set.Size())
	assert.True(t, set.Empty())
	assert.ElementsMatch(t, []int{}, set.Members())
}

func TestSetClone(t *testing.T) {
	t.Parallel()

	original := collections.NewSet(1, 2, 3)
	clone := original.Clone()

	// Should be equal but not the same instance
	assert.True(t, original.Equal(clone))
	assert.ElementsMatch(t, original.Members(), clone.Members())

	// Modifying clone should not affect original
	clone.Add(4)
	assert.False(t, original.Equal(clone))
	assert.False(t, original.Contains(4))
	assert.True(t, clone.Contains(4))

	// Modifying original should not affect clone
	original.Remove(1)
	assert.True(t, clone.Contains(1))
	assert.False(t, original.Contains(1))
}

func TestSetSymmetricDifference(t *testing.T) {
	t.Parallel()

	set1 := collections.NewSet(1, 2, 3, 4)
	set2 := collections.NewSet(3, 4, 5, 6)

	result := set1.SymmetricDifference(set2)

	// Should contain elements in either set but not both
	expected := []int{1, 2, 5, 6}
	assert.ElementsMatch(t, expected, result.Members())

	// Original sets should be unchanged
	assert.ElementsMatch(t, []int{1, 2, 3, 4}, set1.Members())
	assert.ElementsMatch(t, []int{3, 4, 5, 6}, set2.Members())

	// Test with empty sets
	empty := collections.NewSet[int]()
	nonEmpty := collections.NewSet(1, 2, 3)

	symDiff1 := empty.SymmetricDifference(nonEmpty)
	assert.ElementsMatch(t, []int{1, 2, 3}, symDiff1.Members())

	symDiff2 := nonEmpty.SymmetricDifference(empty)
	assert.ElementsMatch(t, []int{1, 2, 3}, symDiff2.Members())

	// Test with identical sets
	identical := collections.NewSet(1, 2, 3)
	symDiffIdentical := set1.SymmetricDifference(identical)
	// Should only contain elements unique to set1
	assert.ElementsMatch(t, []int{4}, symDiffIdentical.Members())
}

func TestSetIter(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3)
	result := make([]int, 0, 3)

	for v := range set.Iter() {
		result = append(result, v)
	}

	assert.ElementsMatch(t, []int{1, 2, 3}, result)
}

func TestSetMembers(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(3, 1, 2)
	result := set.Members()

	assert.ElementsMatch(t, []int{1, 2, 3}, result)
}

func TestSetString(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3)
	str := set.String()

	// String representation should contain all elements
	// Use `Contains` since the output order is non-deterministic
	assert.Contains(t, str, "1")
	assert.Contains(t, str, "2")
	assert.Contains(t, str, "3")
}

func TestSetUnion(t *testing.T) {
	t.Parallel()

	set1 := collections.NewSet(1, 2, 3)
	set2 := collections.NewSet(3, 4, 5)

	result := set1.Union(set2)

	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5}, result.Members())

	// Original sets should be unchanged
	assert.ElementsMatch(t, []int{1, 2, 3}, set1.Members())
	assert.ElementsMatch(t, []int{3, 4, 5}, set2.Members())
}

func TestSetIntersection(t *testing.T) {
	t.Parallel()

	set1 := collections.NewSet(1, 2, 3, 4)
	set2 := collections.NewSet(3, 4, 5, 6)

	result := set1.Intersection(set2)

	assert.ElementsMatch(t, []int{3, 4}, result.Members())
}

func TestSetDifference(t *testing.T) {
	t.Parallel()

	set1 := collections.NewSet(1, 2, 3, 4)
	set2 := collections.NewSet(3, 4, 5, 6)

	// Difference: elements in set1 but not in set2
	result := set1.Difference(set2)

	assert.ElementsMatch(t, []int{1, 2}, result.Members())
}

func TestSetDifferenceWithEmptySet(t *testing.T) {
	t.Parallel()

	empty := collections.NewSet[int]()
	nonEmpty := collections.NewSet(1, 2, 3)

	// Difference with empty
	difference1 := empty.Difference(nonEmpty)
	assert.ElementsMatch(t, []int{}, difference1.Members())

	difference2 := nonEmpty.Difference(empty)
	assert.ElementsMatch(t, []int{1, 2, 3}, difference2.Members())
}

func TestSetFilter(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3, 4, 5, 6)

	// Filter for even numbers
	result := set.Filter(func(n int) bool { return n%2 == 0 })

	assert.ElementsMatch(t, []int{2, 4, 6}, result.Members())
}

func TestTransformSet(t *testing.T) {
	t.Parallel()

	set := collections.NewSet(1, 2, 3)

	// Transform integers to strings
	result := collections.TransformSet(set, func(n int) string {
		return []string{"zero", "one", "two", "three"}[n]
	})

	assert.ElementsMatch(t, []string{"one", "two", "three"}, result.Members())
}

func TestSetOperationsWithEmptySet(t *testing.T) {
	t.Parallel()

	empty := collections.NewSet[int]()
	nonEmpty := collections.NewSet(1, 2, 3)

	// Union with empty
	union1 := empty.Union(nonEmpty)
	assert.ElementsMatch(t, []int{1, 2, 3}, union1.Members())

	union2 := nonEmpty.Union(empty)
	assert.ElementsMatch(t, []int{1, 2, 3}, union2.Members())

	// Intersection with empty
	intersection1 := empty.Intersection(nonEmpty)
	assert.ElementsMatch(t, []int{}, intersection1.Members())

	intersection2 := nonEmpty.Intersection(empty)
	assert.ElementsMatch(t, []int{}, intersection2.Members())
}

func TestSetWithStrings(t *testing.T) {
	t.Parallel()

	set := collections.NewSet("apple", "banana", "cherry")
	assert.ElementsMatch(t, []string{"apple", "banana", "cherry"}, set.Members())

	set.Remove("banana")
	assert.ElementsMatch(t, []string{"apple", "cherry"}, set.Members())
}

func BenchmarkSetAdd(b *testing.B) {
	set := collections.NewSet[int]()

	for i := 0; b.Loop(); i++ {
		set.Add(i)
	}
}

func BenchmarkSetContains(b *testing.B) {
	set := collections.NewSet[int]()
	for i := range 1000 {
		set.Add(i)
	}

	for i := 0; b.Loop(); i++ {
		set.Contains(i % 1000)
	}
}

func BenchmarkSetUnion(b *testing.B) {
	set1 := collections.NewSet[int]()
	set2 := collections.NewSet[int]()

	for i := range 500 {
		set1.Add(i)
		set2.Add(i + 250) // Some overlap
	}

	for b.Loop() {
		_ = set1.Union(set2)
	}
}

func BenchmarkSetIntersection(b *testing.B) {
	set1 := collections.NewSet[int]()
	set2 := collections.NewSet[int]()

	for i := range 500 {
		set1.Add(i)
		set2.Add(i + 250) // Some overlap
	}

	for b.Loop() {
		_ = set1.Intersection(set2)
	}
}

func BenchmarkSetDifference(b *testing.B) {
	set1 := collections.NewSet[int]()
	set2 := collections.NewSet[int]()

	for i := range 500 {
		set1.Add(i)
		set2.Add(i + 250) // Some overlap
	}

	for b.Loop() {
		_ = set1.Difference(set2)
	}
}
