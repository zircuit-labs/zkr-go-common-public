package iter_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zkriter "github.com/zircuit-labs/zkr-go-common/iter"
)

func TestFilter_Integers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     []int
		predicate zkriter.Predicate[int]
		expected  []int
	}{
		{
			name:      "filter even numbers",
			input:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			predicate: func(n int) bool { return n%2 == 0 },
			expected:  []int{2, 4, 6, 8, 10},
		},
		{
			name:      "filter odd numbers",
			input:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			predicate: func(n int) bool { return n%2 != 0 },
			expected:  []int{1, 3, 5, 7, 9},
		},
		{
			name:      "empty sequence",
			input:     []int{},
			predicate: func(n int) bool { return true },
			expected:  nil,
		},
		{
			name:      "all elements filtered out",
			input:     []int{1, 3, 5, 7, 9},
			predicate: func(n int) bool { return n%2 == 0 },
			expected:  nil,
		},
		{
			name:      "all elements pass filter",
			input:     []int{2, 4, 6, 8, 10},
			predicate: func(n int) bool { return n%2 == 0 },
			expected:  []int{2, 4, 6, 8, 10},
		},
		{
			name:      "greater than threshold",
			input:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			predicate: func(n int) bool { return n > 5 },
			expected:  []int{6, 7, 8, 9, 10},
		},
		{
			name:      "multiples of three",
			input:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			predicate: func(n int) bool { return n%3 == 0 },
			expected:  []int{3, 6, 9, 12},
		},
		{
			name:      "single element passes",
			input:     []int{1, 2, 3},
			predicate: func(n int) bool { return n == 2 },
			expected:  []int{2},
		},
		{
			name:      "single element input passes",
			input:     []int{42},
			predicate: func(n int) bool { return n == 42 },
			expected:  []int{42},
		},
		{
			name:      "single element input fails",
			input:     []int{42},
			predicate: func(n int) bool { return n == 0 },
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filtered := zkriter.Filter(tt.predicate, slices.Values(tt.input))
			result := slices.Collect(filtered)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilter_Strings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     []string
		predicate zkriter.Predicate[string]
		expected  []string
	}{
		{
			name:      "filter by length greater than 2",
			input:     []string{"a", "ab", "abc", "abcd", "abcde"},
			predicate: func(s string) bool { return len(s) > 2 },
			expected:  []string{"abc", "abcd", "abcde"},
		},
		{
			name:      "filter empty strings",
			input:     []string{"", "a", "", "b", "c", ""},
			predicate: func(s string) bool { return s != "" },
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "filter strings starting with 'a'",
			input:     []string{"apple", "banana", "apricot", "cherry", "avocado"},
			predicate: func(s string) bool { return s != "" && s[0] == 'a' },
			expected:  []string{"apple", "apricot", "avocado"},
		},
		{
			name:      "all strings pass",
			input:     []string{"go", "is", "fun"},
			predicate: func(s string) bool { return len(s) >= 2 },
			expected:  []string{"go", "is", "fun"},
		},
		{
			name:      "empty input",
			input:     []string{},
			predicate: func(s string) bool { return true },
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filtered := zkriter.Filter(tt.predicate, slices.Values(tt.input))
			result := slices.Collect(filtered)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilter_CustomTypes(t *testing.T) {
	t.Parallel()
	type Person struct {
		Name string
		Age  int
	}

	tests := []struct {
		name      string
		input     []Person
		predicate zkriter.Predicate[Person]
		expected  []Person
	}{
		{
			name: "filter adults",
			input: []Person{
				{"Alice", 25},
				{"Bob", 30},
				{"Charlie", 17},
				{"David", 45},
				{"Eve", 19},
			},
			predicate: func(p Person) bool { return p.Age >= 21 },
			expected: []Person{
				{"Alice", 25},
				{"Bob", 30},
				{"David", 45},
			},
		},
		{
			name: "filter by name length",
			input: []Person{
				{"Al", 25},
				{"Bob", 30},
				{"Charlie", 35},
				{"Jo", 40},
			},
			predicate: func(p Person) bool { return len(p.Name) > 3 },
			expected: []Person{
				{"Charlie", 35},
			},
		},
		{
			name: "complex condition",
			input: []Person{
				{"Alice", 25},
				{"Bob", 18},
				{"Charlie", 30},
				{"David", 16},
				{"Eve", 22},
			},
			predicate: func(p Person) bool { return p.Age >= 18 && len(p.Name) <= 5 },
			expected: []Person{
				{"Alice", 25},
				{"Bob", 18},
				{"Eve", 22},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filtered := zkriter.Filter(tt.predicate, slices.Values(tt.input))
			result := slices.Collect(filtered)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilter_Pointers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     []*int
		predicate zkriter.Predicate[*int]
		expected  []int // expected values (not pointers) for easier comparison
	}{
		{
			name:      "filter nil values",
			input:     []*int{nil, intPtr(1), nil, intPtr(2), intPtr(3), nil},
			predicate: func(p *int) bool { return p != nil },
			expected:  []int{1, 2, 3},
		},
		{
			name:      "filter non-nil even values",
			input:     []*int{nil, intPtr(1), intPtr(2), nil, intPtr(3), intPtr(4)},
			predicate: func(p *int) bool { return p != nil && *p%2 == 0 },
			expected:  []int{2, 4},
		},
		{
			name:      "all nil",
			input:     []*int{nil, nil, nil},
			predicate: func(p *int) bool { return p != nil },
			expected:  nil,
		},
		{
			name:      "no nil values",
			input:     []*int{intPtr(1), intPtr(2), intPtr(3)},
			predicate: func(p *int) bool { return p != nil },
			expected:  []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filtered := zkriter.Filter(tt.predicate, slices.Values(tt.input))
			result := slices.Collect(filtered)

			require.Len(t, result, len(tt.expected))
			for i, v := range result {
				require.NotNil(t, v)
				assert.Equal(t, tt.expected[i], *v)
			}
		})
	}
}

func TestFilter_EarlyTermination(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     []int
		predicate zkriter.Predicate[int]
		takeCount int
		expected  []int
	}{
		{
			name:      "take first 3 even numbers",
			input:     []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			predicate: func(n int) bool { return n%2 == 0 },
			takeCount: 3,
			expected:  []int{2, 4, 6},
		},
		{
			name:      "take more than available",
			input:     []int{1, 2, 3, 4, 5},
			predicate: func(n int) bool { return n%2 == 0 },
			takeCount: 10,
			expected:  []int{2, 4},
		},
		{
			name:      "take zero elements",
			input:     []int{1, 2, 3, 4, 5},
			predicate: func(n int) bool { return n%2 == 0 },
			takeCount: 0,
			expected:  nil,
		},
		{
			name:      "take one element",
			input:     []int{1, 2, 3, 4, 5},
			predicate: func(n int) bool { return n > 0 },
			takeCount: 1,
			expected:  []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filtered := zkriter.Filter(tt.predicate, slices.Values(tt.input))

			var result []int
			count := 0
			for v := range filtered {
				if count >= tt.takeCount {
					break
				}
				result = append(result, v)
				count++
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilter_StatefulPredicate(t *testing.T) {
	t.Parallel()
	t.Run("filter first N matching elements", func(t *testing.T) {
		t.Parallel()
		input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		count := 0
		maxCount := 3

		predicate := func(n int) bool {
			if n%2 == 0 && count < maxCount {
				count++
				return true
			}
			return false
		}

		filtered := zkriter.Filter(predicate, slices.Values(input))
		result := slices.Collect(filtered)

		expected := []int{2, 4, 6}
		assert.Equal(t, expected, result)
	})

	t.Run("accumulating sum filter", func(t *testing.T) {
		t.Parallel()
		input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		sum := 0
		maxSum := 10

		predicate := func(n int) bool {
			if sum+n <= maxSum {
				sum += n
				return true
			}
			return false
		}

		filtered := zkriter.Filter(predicate, slices.Values(input))
		result := slices.Collect(filtered)

		expected := []int{1, 2, 3, 4} // 1+2+3+4 = 10
		assert.Equal(t, expected, result)
	})
}

func intPtr(n int) *int {
	return &n
}

func BenchmarkFilter(b *testing.B) {
	benchmarks := []struct {
		name      string
		size      int
		predicate zkriter.Predicate[int]
	}{
		{
			name:      "even_numbers_small",
			size:      100,
			predicate: func(n int) bool { return n%2 == 0 },
		},
		{
			name:      "even_numbers_medium",
			size:      1000,
			predicate: func(n int) bool { return n%2 == 0 },
		},
		{
			name:      "even_numbers_large",
			size:      10000,
			predicate: func(n int) bool { return n%2 == 0 },
		},
		{
			name: "complex_predicate_medium",
			size: 1000,
			predicate: func(n int) bool {
				if n < 2 {
					return false
				}
				if n == 2 {
					return true
				}
				if n%2 == 0 {
					return false
				}
				return n%3 != 0 && n%5 != 0
			},
		},
		{
			name:      "all_pass_medium",
			size:      1000,
			predicate: func(n int) bool { return true },
		},
		{
			name:      "none_pass_medium",
			size:      1000,
			predicate: func(n int) bool { return false },
		},
	}

	for _, bm := range benchmarks {
		input := make([]int, bm.size)
		for i := range input {
			input[i] = i
		}

		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				filtered := zkriter.Filter(bm.predicate, slices.Values(input))
				_ = slices.Collect(filtered)
			}
		})
	}
}

func BenchmarkFilterNoCollection(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		input := make([]int, size)
		for i := range input {
			input[i] = i
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			isEven := func(n int) bool { return n%2 == 0 }
			b.ResetTimer()
			for b.Loop() {
				filtered := zkriter.Filter(isEven, slices.Values(input))
				count := 0
				for range filtered {
					count++
				}
			}
		})
	}
}
