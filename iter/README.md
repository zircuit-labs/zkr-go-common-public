# iter

The `iter` package provides functional operations for Go 1.23+ iterators, enabling composable and lazy sequence transformations.

## Overview

This package leverages Go 1.23's new iterator support to provide functional programming patterns for sequence manipulation. All operations are lazy-evaluated, meaning transformations are only applied when elements are consumed.

## Functions

### Filter

Filters elements from a sequence based on a predicate function.

```go
type Predicate[V any] func(V) bool

func Filter[V any](p Predicate[V], s iter.Seq[V]) iter.Seq[V]
```

**Example:**

```go
import (
    "slices"
    "github.com/zircuit-labs/zkr-go-common/iter"
)

// Filter even numbers
numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
isEven := func(n int) bool { return n%2 == 0 }

filtered := iter.Filter(isEven, slices.Values(numbers))
result := slices.Collect(filtered) // [2, 4, 6, 8, 10]
```

### Transform

Applies a transformation function to each element in a sequence, converting from one type to another.

```go
type Transformation[S, T any] func(S) T

func Transform[S, T any](t Transformation[S, T], s iter.Seq[S]) iter.Seq[T]
```

**Example:**

```go
import (
    "slices"
    "strconv"
    "github.com/zircuit-labs/zkr-go-common/iter"
)

// Convert integers to strings
numbers := []int{1, 2, 3, 4, 5}
toString := func(n int) string { return strconv.Itoa(n) }

transformed := iter.Transform(toString, slices.Values(numbers))
result := slices.Collect(transformed) // ["1", "2", "3", "4", "5"]
```

### And

Returns true if the predicate returns true for **all** elements in the sequence.

```go
func And[V any](p Predicate[V], s iter.Seq[V]) bool
```

**Example:**

```go
import (
    "slices"
    "github.com/zircuit-labs/zkr-go-common/iter"
)

// Check if all numbers are positive
numbers := []int{1, 5, 10, 25}
allPositive := iter.And(func(n int) bool { return n > 0 }, slices.Values(numbers))
// allPositive: true

// Check if all numbers are even
allEven := iter.And(func(n int) bool { return n%2 == 0 }, slices.Values(numbers))
// allEven: false (because 1, 5, and 25 are odd)
```

### Or

Returns true if the predicate returns true for **any** element in the sequence.

```go
func Or[V any](p Predicate[V], s iter.Seq[V]) bool
```

**Example:**

```go
import (
    "slices"
    "github.com/zircuit-labs/zkr-go-common/iter"
)

// Check if any numbers are even
numbers := []int{1, 3, 4, 7}
anyEven := iter.Or(func(n int) bool { return n%2 == 0 }, slices.Values(numbers))
// anyEven: true (because 4 is even)

// Check if any numbers are negative
anyNegative := iter.Or(func(n int) bool { return n < 0 }, slices.Values(numbers))
// anyNegative: false (all numbers are positive)
```

## Composition

Functions can be chained together for complex transformations:

```go
// Transform integers to strings, then filter
numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// First transform to strings
toString := func(n int) string { return strconv.Itoa(n) }
transformed := iter.Transform(toString, slices.Values(numbers))

// Then filter for even number strings
isEvenString := func(s string) bool {
    n, _ := strconv.Atoi(s)
    return n%2 == 0
}
filtered := iter.Filter(isEvenString, transformed)

result := slices.Collect(filtered) // ["2", "4", "6", "8", "10"]
```

### Logical Operations with Composition

The logical functions work seamlessly with other iterator operations:

```go
// Check if all filtered elements meet a condition
numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
evenNumbers := iter.Filter(func(n int) bool { return n%2 == 0 }, slices.Values(numbers))

// Check if all even numbers are less than 20
allSmallEvens := iter.And(func(n int) bool { return n < 20 }, evenNumbers)
// allSmallEvens: true

// Check if any even numbers are greater than 5
anyLargeEvens := iter.Or(func(n int) bool { return n > 5 }, evenNumbers)
// anyLargeEvens: true (6, 8, 10 are > 5)
```

## Early Termination

All operations support early termination when the consumer stops requesting elements:

```go
numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
isEven := func(n int) bool { return n%2 == 0 }

filtered := iter.Filter(isEven, slices.Values(numbers))

// Only take first 3 even numbers
var result []int
for v := range filtered {
    result = append(result, v)
    if len(result) >= 3 {
        break
    }
}
// result: [2, 4, 6]
```

### Logical Functions Early Termination

The `And` and `Or` functions are particularly efficient due to short-circuiting:

```go
// And stops as soon as it finds the first false result
numbers := []int{2, 4, 5, 8, 10} // 5 is odd
allEven := iter.And(func(n int) bool { return n%2 == 0 }, slices.Values(numbers))
// Stops after checking 2, 4, 5 (doesn't check 8, 10)

// Or stops as soon as it finds the first true result
numbers := []int{1, 3, 4, 7, 9} // 4 is even
anyEven := iter.Or(func(n int) bool { return n%2 == 0 }, slices.Values(numbers))
// Stops after checking 1, 3, 4 (doesn't check 7, 9)
```

## Performance

All operations are lazy-evaluated, meaning:

- No intermediate collections are created
- Transformations are only applied to consumed elements
- Memory usage is constant regardless of input size
- Early termination is efficient

## Requirements

- Go 1.23 or later (for iterator support)

## Testing

The package includes comprehensive tests with parallel execution support:

```bash
go test ./iter/...
```

Run benchmarks:

```bash
go test -bench=. ./iter/...
```
