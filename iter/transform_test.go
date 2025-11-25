package iter_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	zkriter "github.com/zircuit-labs/zkr-go-common/iter"
)

func TestTransform_IntToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          []int
		transformation zkriter.Transformation[int, string]
		expected       []string
	}{
		{
			name:           "convert integers to strings",
			input:          []int{1, 2, 3, 4, 5},
			transformation: strconv.Itoa,
			expected:       []string{"1", "2", "3", "4", "5"},
		},
		{
			name:           "convert with custom format",
			input:          []int{1, 2, 3},
			transformation: func(n int) string { return fmt.Sprintf("num_%d", n) },
			expected:       []string{"num_1", "num_2", "num_3"},
		},
		{
			name:           "empty input",
			input:          []int{},
			transformation: strconv.Itoa,
			expected:       nil,
		},
		{
			name:           "single element",
			input:          []int{42},
			transformation: func(n int) string { return fmt.Sprintf("[%d]", n) },
			expected:       []string{"[42]"},
		},
		{
			name:           "negative numbers",
			input:          []int{-5, -3, -1, 0, 1, 3, 5},
			transformation: strconv.Itoa,
			expected:       []string{"-5", "-3", "-1", "0", "1", "3", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransform_StringManipulation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          []string
		transformation zkriter.Transformation[string, string]
		expected       []string
	}{
		{
			name:           "uppercase transformation",
			input:          []string{"hello", "world", "test"},
			transformation: strings.ToUpper,
			expected:       []string{"HELLO", "WORLD", "TEST"},
		},
		{
			name:           "lowercase transformation",
			input:          []string{"HELLO", "World", "TeSt"},
			transformation: strings.ToLower,
			expected:       []string{"hello", "world", "test"},
		},
		{
			name:           "trim spaces",
			input:          []string{"  hello  ", "world  ", "  test"},
			transformation: strings.TrimSpace,
			expected:       []string{"hello", "world", "test"},
		},
		{
			name:           "add prefix",
			input:          []string{"a", "b", "c"},
			transformation: func(s string) string { return "prefix_" + s },
			expected:       []string{"prefix_a", "prefix_b", "prefix_c"},
		},
		{
			name:           "empty strings",
			input:          []string{"", "", ""},
			transformation: func(s string) string { return "empty" },
			expected:       []string{"empty", "empty", "empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransform_StructTypes(t *testing.T) {
	t.Parallel()

	type Person struct {
		Name string
		Age  int
	}

	type PersonDTO struct {
		FullName string
		Years    int
	}

	tests := []struct {
		name           string
		input          []Person
		transformation zkriter.Transformation[Person, PersonDTO]
		expected       []PersonDTO
	}{
		{
			name: "transform person to DTO",
			input: []Person{
				{"Alice", 30},
				{"Bob", 25},
				{"Charlie", 35},
			},
			transformation: func(p Person) PersonDTO {
				return PersonDTO{FullName: p.Name, Years: p.Age}
			},
			expected: []PersonDTO{
				{"Alice", 30},
				{"Bob", 25},
				{"Charlie", 35},
			},
		},
		{
			name: "transform with modification",
			input: []Person{
				{"Alice", 30},
				{"Bob", 25},
			},
			transformation: func(p Person) PersonDTO {
				return PersonDTO{FullName: "Mr/Ms " + p.Name, Years: p.Age + 1}
			},
			expected: []PersonDTO{
				{"Mr/Ms Alice", 31},
				{"Mr/Ms Bob", 26},
			},
		},
		{
			name:           "empty input",
			input:          []Person{},
			transformation: func(p Person) PersonDTO { return PersonDTO{} },
			expected:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransform_NumericOperations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          []float64
		transformation zkriter.Transformation[float64, int]
		expected       []int
	}{
		{
			name:           "round to int",
			input:          []float64{1.2, 2.7, 3.5, 4.9},
			transformation: func(f float64) int { return int(f) },
			expected:       []int{1, 2, 3, 4},
		},
		{
			name:           "multiply and convert",
			input:          []float64{1.5, 2.5, 3.5},
			transformation: func(f float64) int { return int(f * 10) },
			expected:       []int{15, 25, 35},
		},
		{
			name:           "negative values",
			input:          []float64{-1.5, -2.5, -3.5},
			transformation: func(f float64) int { return int(f) },
			expected:       []int{-1, -2, -3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransform_SliceToElement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          [][]int
		transformation zkriter.Transformation[[]int, int]
		expected       []int
	}{
		{
			name:  "sum of elements",
			input: [][]int{{1, 2, 3}, {4, 5}, {6}},
			transformation: func(slice []int) int {
				sum := 0
				for _, v := range slice {
					sum += v
				}
				return sum
			},
			expected: []int{6, 9, 6},
		},
		{
			name:           "length of slice",
			input:          [][]int{{1, 2, 3}, {4, 5}, {}, {6, 7, 8, 9}},
			transformation: func(slice []int) int { return len(slice) },
			expected:       []int{3, 2, 0, 4},
		},
		{
			name:  "first element or zero",
			input: [][]int{{10, 20}, {30}, {}, {40, 50}},
			transformation: func(slice []int) int {
				if len(slice) > 0 {
					return slice[0]
				}
				return 0
			},
			expected: []int{10, 30, 0, 40},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransform_EarlyTermination(t *testing.T) {
	t.Parallel()

	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	double := func(n int) int { return n * 2 }

	transformed := zkriter.Transform(double, slices.Values(input))

	result := make([]int, 0, 5)
	for v := range transformed {
		result = append(result, v)
		if len(result) >= 5 {
			break
		}
	}

	expected := []int{2, 4, 6, 8, 10}
	assert.Equal(t, expected, result)
}

func TestTransform_ChainedWithFilter(t *testing.T) {
	t.Parallel()

	// Transform integers to strings, then filter
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// First transform to strings
	transformed := zkriter.Transform(strconv.Itoa, slices.Values(input))

	// Then filter for even number strings
	isEvenString := func(s string) bool {
		n, _ := strconv.Atoi(s)
		return n%2 == 0
	}
	filtered := zkriter.Filter(isEvenString, transformed)

	result := slices.Collect(filtered)
	expected := []string{"2", "4", "6", "8", "10"}
	assert.Equal(t, expected, result)
}

func TestTransform_Pointers(t *testing.T) {
	t.Parallel()

	intPtr := func(n int) *int { return &n }

	tests := []struct {
		name           string
		input          []*int
		transformation zkriter.Transformation[*int, int]
		expected       []int
	}{
		{
			name:  "dereference non-nil pointers",
			input: []*int{intPtr(1), intPtr(2), intPtr(3)},
			transformation: func(p *int) int {
				if p != nil {
					return *p
				}
				return 0
			},
			expected: []int{1, 2, 3},
		},
		{
			name:  "handle nil pointers",
			input: []*int{intPtr(1), nil, intPtr(3), nil, intPtr(5)},
			transformation: func(p *int) int {
				if p != nil {
					return *p
				}
				return -1
			},
			expected: []int{1, -1, 3, -1, 5},
		},
		{
			name:  "all nil",
			input: []*int{nil, nil, nil},
			transformation: func(p *int) int {
				if p != nil {
					return *p
				}
				return 0
			},
			expected: []int{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformed := zkriter.Transform(tt.transformation, slices.Values(tt.input))
			result := slices.Collect(transformed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkTransform(b *testing.B) {
	benchmarks := []struct {
		name string
		size int
	}{
		{"small_100", 100},
		{"medium_1000", 1000},
		{"large_10000", 10000},
	}

	for _, bm := range benchmarks {
		input := make([]int, bm.size)
		for i := range input {
			input[i] = i
		}

		b.Run(bm.name+"_int_to_string", func(b *testing.B) {
			for b.Loop() {
				transformed := zkriter.Transform(strconv.Itoa, slices.Values(input))
				_ = slices.Collect(transformed)
			}
		})

		b.Run(bm.name+"_arithmetic", func(b *testing.B) {
			double := func(n int) int { return n * 2 }
			for b.Loop() {
				transformed := zkriter.Transform(double, slices.Values(input))
				_ = slices.Collect(transformed)
			}
		})
	}
}

func BenchmarkTransformNoCollection(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		input := make([]int, size)
		for i := range input {
			input[i] = i
		}

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			double := func(n int) int { return n * 2 }
			for b.Loop() {
				transformed := zkriter.Transform(double, slices.Values(input))
				count := 0
				for range transformed {
					count++
				}
			}
		})
	}
}
