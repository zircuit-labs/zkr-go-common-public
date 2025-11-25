package iter_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	zkriter "github.com/zircuit-labs/zkr-go-common/iter"
)

func TestAnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		values    []int
		predicate func(int) bool
		expected  bool
	}{
		{
			name:      "all elements satisfy predicate",
			values:    []int{2, 4, 6, 8},
			predicate: func(n int) bool { return n%2 == 0 }, // even numbers
			expected:  true,
		},
		{
			name:      "not all elements satisfy predicate",
			values:    []int{2, 4, 5, 8},
			predicate: func(n int) bool { return n%2 == 0 }, // even numbers
			expected:  false,
		},
		{
			name:      "empty sequence",
			values:    []int{},
			predicate: func(n int) bool { return n > 0 },
			expected:  true, // vacuous truth
		},
		{
			name:      "single element satisfies",
			values:    []int{10},
			predicate: func(n int) bool { return n > 5 },
			expected:  true,
		},
		{
			name:      "single element does not satisfy",
			values:    []int{3},
			predicate: func(n int) bool { return n > 5 },
			expected:  false,
		},
		{
			name:      "all positive numbers",
			values:    []int{1, 5, 10, 25},
			predicate: func(n int) bool { return n > 0 },
			expected:  true,
		},
		{
			name:      "contains negative number",
			values:    []int{1, 5, -1, 25},
			predicate: func(n int) bool { return n > 0 },
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := zkriter.And(tt.predicate, slices.Values(tt.values))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		values    []int
		predicate func(int) bool
		expected  bool
	}{
		{
			name:      "at least one element satisfies predicate",
			values:    []int{1, 3, 4, 7},
			predicate: func(n int) bool { return n%2 == 0 }, // even numbers
			expected:  true,
		},
		{
			name:      "no elements satisfy predicate",
			values:    []int{1, 3, 5, 7},
			predicate: func(n int) bool { return n%2 == 0 }, // even numbers
			expected:  false,
		},
		{
			name:      "empty sequence",
			values:    []int{},
			predicate: func(n int) bool { return n > 0 },
			expected:  false,
		},
		{
			name:      "single element satisfies",
			values:    []int{10},
			predicate: func(n int) bool { return n > 5 },
			expected:  true,
		},
		{
			name:      "single element does not satisfy",
			values:    []int{3},
			predicate: func(n int) bool { return n > 5 },
			expected:  false,
		},
		{
			name:      "first element satisfies",
			values:    []int{10, 2, 3, 4},
			predicate: func(n int) bool { return n > 5 },
			expected:  true,
		},
		{
			name:      "last element satisfies",
			values:    []int{2, 3, 4, 10},
			predicate: func(n int) bool { return n > 5 },
			expected:  true,
		},
		{
			name:      "multiple elements satisfy",
			values:    []int{10, 3, 8, 4},
			predicate: func(n int) bool { return n > 5 },
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := zkriter.Or(tt.predicate, slices.Values(tt.values))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAndWithStrings(t *testing.T) {
	t.Parallel()

	words := []string{"hello", "world", "test"}

	// All words have length > 3
	result := zkriter.And(func(s string) bool { return len(s) > 3 }, slices.Values(words))
	assert.True(t, result)

	// Not all words start with 'h'
	result = zkriter.And(func(s string) bool { return s[0] == 'h' }, slices.Values(words))
	assert.False(t, result)
}

func TestOrWithStrings(t *testing.T) {
	t.Parallel()

	words := []string{"hello", "world", "test"}

	// At least one word starts with 'h'
	result := zkriter.Or(func(s string) bool { return s[0] == 'h' }, slices.Values(words))
	assert.True(t, result)

	// No words start with 'z'
	result = zkriter.Or(func(s string) bool { return s[0] == 'z' }, slices.Values(words))
	assert.False(t, result)
}

func TestAndEarlyTermination(t *testing.T) {
	t.Parallel()

	callCount := 0
	predicate := func(n int) bool {
		callCount++
		return n%2 == 0 // even numbers
	}

	values := []int{2, 4, 5, 8, 10} // First odd number is at index 2
	result := zkriter.And(predicate, slices.Values(values))

	assert.False(t, result)
	assert.Equal(t, 3, callCount) // Should stop after checking 2, 4, 5
}

func TestOrEarlyTermination(t *testing.T) {
	t.Parallel()

	callCount := 0
	predicate := func(n int) bool {
		callCount++
		return n%2 == 0 // even numbers
	}

	values := []int{1, 3, 4, 8, 10} // First even number is at index 2
	result := zkriter.Or(predicate, slices.Values(values))

	assert.True(t, result)
	assert.Equal(t, 3, callCount) // Should stop after checking 1, 3, 4
}
