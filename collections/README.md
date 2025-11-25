# collections

The `collections` package provides generic data structures that complement Go's standard library collections. These types are designed to be memory-efficient, type-safe, and integrate seamlessly with Go's iterators and functional programming patterns.

## Overview

This package offers:

- **Type-safe generic collections** - Leverage Go's generics for compile-time safety
- **Iterator integration** - Full support for Go 1.23+ iterators (`iter.Seq`)
- **Functional programming** - Filter, transform, and compose operations
- **Memory efficiency** - Optimized internal representations
- **Standard library compatibility** - Works with `slices`, `maps`, and other stdlib packages

## Available Collections

### Set[E comparable]

A mathematical set implementation for comparable types, backed by a map for O(1) operations.

```go
import "github.com/zircuit-labs/zkr-go-common/collections"

// Create sets
numbers := collections.NewSet(1, 2, 3, 4, 5)
words := collections.NewSet("apple", "banana", "cherry")

// Basic operations
numbers.Add(6, 7)
numbers.Remove(1)
fmt.Println(numbers.Contains(3)) // true
fmt.Println(numbers.Size())      // 6

// Set operations
evens := collections.NewSet(2, 4, 6, 8)
union := numbers.Union(evens)
intersection := numbers.Intersection(evens)

// Functional operations
filtered := numbers.Filter(func(n int) bool { return n > 3 })
strings := collections.TransformSet(numbers, func(n int) string {
    return fmt.Sprintf("num_%d", n)
})
```

## Key Features

### Iterator Support

All collections support Go 1.23+ iterators for seamless integration:

```go
set := collections.NewSet(1, 2, 3, 4, 5)

// Iterate over elements
for value := range set.Iter() {
    fmt.Println(value)
}

// Add from iterator
moreNumbers := []int{6, 7, 8}
set.AddIter(slices.Values(moreNumbers))

// Remove from iterator
toRemove := []int{1, 2}
set.RemoveIter(slices.Values(toRemove))
```

### Functional Programming

Collections integrate with the `iter` package for functional operations:

```go
import zkriter "github.com/zircuit-labs/zkr-go-common/iter"

numbers := collections.NewSet(1, 2, 3, 4, 5, 6)

// Chain functional operations
result := collections.TransformSet(
    numbers.Filter(func(n int) bool { return n%2 == 0 }), // Even numbers
    func(n int) string { return fmt.Sprintf("even_%d", n) }, // To strings
)
// result contains: {"even_2", "even_4", "even_6"}
```

### Memory Efficiency

Collections are optimized for memory usage:

- **Set** uses `map[E]struct{}` - zero-byte values minimize memory overhead
- **Bulk operations** use iterators to avoid intermediate allocations
- **In-place modifications** where possible to reduce garbage collection pressure

## Performance

The collections are designed for high performance:

- **Set operations**: O(1) for Contains, Add, Remove
- **Set Union/Intersection**: O(n) where n is the size of the smaller set
- **Iterator operations**: Lazy evaluation prevents unnecessary allocations
- **Bulk operations**: Optimized batch processing

### Benchmarks

```go
// Example benchmark results (your results may vary)
BenchmarkSetAdd-8           1000000000    0.5 ns/op
BenchmarkSetContains-8      2000000000    0.3 ns/op
BenchmarkSetUnion-8         5000000       250 ns/op
BenchmarkSetIntersection-8  3000000       400 ns/op
```

## Type Constraints

Collections use appropriate type constraints:

- **comparable** - For basic set operations (required for map keys)

```go
// Works with any comparable type
intSet := collections.NewSet(1, 2, 3)
stringSet := collections.NewSet("a", "b", "c")
structSet := collections.NewSet(Point{1, 2}, Point{3, 4})
```

## Integration

The collections package integrates well with other packages in this repository:

- **iter** - Functional operations (Filter, Transform)
- **Standard library** - slices, maps, fmt packages
- **Testing** - Comprehensive test coverage with benchmarks

## Future Collections

The collections package is designed to accommodate additional data structures in the future as needed.

## Best Practices

1. **Leverage iterators** - Use `AddIter`/`RemoveIter` for bulk operations
2. **Functional composition** - Chain Filter and Transform for complex operations
3. **Memory awareness** - Prefer bulk operations over individual calls in loops
4. **Type constraints** - Use the most specific constraint for your use case

## Examples

### Deduplication

```go
func dedupe[E comparable](slice []E) []E {
    set := collections.NewSet(slice...)
    return set.Members()
}
```

### Set Operations

```go
func findCommonElements[E comparable](a, b []E) []E {
    setA := collections.NewSet(a...)
    setB := collections.NewSet(b...)
    return setA.Intersection(setB).Members()
}
```

### Functional Pipeline

```go
numbers := collections.NewSet(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

// Find squares of even numbers greater than 4
result := collections.TransformSet(
    numbers.Filter(func(n int) bool { return n > 4 && n%2 == 0 }),
    func(n int) int { return n * n },
)
// result: {36, 64, 100}

## Dependencies

- Go 1.23+ (for iterator support)
- `github.com/zircuit-labs/zkr-go-common/iter` (functional operations)
- Standard library: `fmt`, `iter`, `maps`, `slices`

## Performance Notes

- Set operations are optimized for the common case
- Iterator-based operations use lazy evaluation
- Bulk operations are preferred over individual operations in loops
- Memory allocation is minimized through careful API design
