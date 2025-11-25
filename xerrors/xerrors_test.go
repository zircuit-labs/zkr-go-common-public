package xerrors_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zircuit-labs/zkr-go-common/xerrors"
)

var errTest = fmt.Errorf("this is a test error")

func wrap(err error) error {
	return fmt.Errorf("wrapping: %w", err)
}

func TestExtendedError(t *testing.T) {
	t.Parallel()

	type dataOne struct {
		s1 string
		s2 string
	}

	type dataTwo struct {
		t time.Time
		i int
	}

	type dataThree struct{}

	d1 := dataOne{
		s1: "hello",
		s2: "world",
	}

	d2 := dataTwo{
		t: time.Now(),
		i: 17,
	}

	// extending nil is still nil
	e0 := xerrors.Extend(d1, nil)
	if e0 != nil {
		t.Errorf("unexpected error: want: %v, got %v", nil, e0)
	}

	// extending errTest with d1 is still an errTest
	e1 := xerrors.Extend(d1, errTest)
	if !errors.Is(e1, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e1)
	}

	// extending e1 with d2 is still an errTest and an e1
	e2 := xerrors.Extend(d2, e1)
	if !errors.Is(e2, e1) {
		t.Errorf("unmatched error: want: %v, got %v", e1, e2)
	}
	if !errors.Is(e2, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e2)
	}

	// wrapping e2 (twice) is still an errTest, e1, and e2
	e3 := wrap(wrap(e2))
	if !errors.Is(e3, e2) {
		t.Errorf("unmatched error: want: %v, got %v", e2, e3)
	}
	if !errors.Is(e3, e1) {
		t.Errorf("unmatched error: want: %v, got %v", e1, e3)
	}
	if !errors.Is(e3, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e3)
	}

	// able to extract the data from e3 that was added to e1
	f1, ok := xerrors.Extract[dataOne](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d1 != f1 {
		t.Errorf("expected equal values: want %v, got %v", d1, f1)
	}

	// able to extract the data from e3 that was added to e2
	f2, ok := xerrors.Extract[dataTwo](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d2 != f2 {
		t.Errorf("expected equal values: want %v, got %v", d2, f2)
	}

	// properly fails to extract data that was never added
	_, ok = xerrors.Extract[dataThree](e3)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}
}

func TestExtendedWithSameType(t *testing.T) {
	t.Parallel()

	type dataOne struct {
		s1 string
		s2 string
	}

	d1 := dataOne{
		s1: "hello",
		s2: "world",
	}

	d2 := dataOne{
		s1: "goodbye",
		s2: "friend",
	}

	// extending an error with the same data type is fine
	// but extracting it will only give the outer-most (ie last extended) value
	e1 := xerrors.Extend(d1, errTest)
	e2 := xerrors.Extend(d2, e1)

	f1, ok := xerrors.Extract[dataOne](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d2 != f1 {
		t.Errorf("expected equal values: want %v, got %v", d2, f1)
	}

	// however if unwrap manually, the data is still there and accessible
	e3 := errors.Unwrap(e2)

	f2, ok := xerrors.Extract[dataOne](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d1 != f2 {
		t.Errorf("expected equal values: want %v, got %v", d1, f2)
	}
}

type (
	ClassA int
	ClassB int
)

const (
	AZero ClassA = iota
	AOne
	ATwo

	BZero ClassB = iota
	BOne
)

func TestExtendedWithMultipleTypedefs(t *testing.T) {
	t.Parallel()

	// ClassA and ClassB are different types but both are int under the hood
	// This test proves that Extract can tell the difference as expected
	e1 := xerrors.Extend(ATwo, errTest)
	e2 := xerrors.Extend(BOne, e1)

	// e2 has a ClassA of ATwo
	f1, ok := xerrors.Extract[ClassA](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if f1 != ATwo {
		t.Errorf("expected equal values: want %v, got %v", ATwo, f1)
	}

	// e2 also has a ClassB of BOne
	f2, ok := xerrors.Extract[ClassB](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if f2 != BOne {
		t.Errorf("expected equal values: want %v, got %v", BOne, f2)
	}

	// e1 was never wrapped with a ClassB
	_, ok = xerrors.Extract[ClassB](e1)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}

	// ClassB didn't have a value defined for 2. Make sure that wasn't why the above passes.
	e3 := xerrors.Extend(AZero, errTest)
	_, ok = xerrors.Extract[ClassB](e3)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}
}

func TestUnjoinNil(t *testing.T) {
	t.Parallel()

	result := xerrors.Unjoin(nil)
	assert.Nil(t, result)
}

func TestUnjoinSingleError(t *testing.T) {
	t.Parallel()

	err := errors.New("error message")
	result := xerrors.Unjoin(err)
	assert.Equal(t, []error{err}, result)
}

func TestUnjoinMultipleErrors(t *testing.T) {
	t.Parallel()

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joinedErr := errors.Join(err1, err2)
	result := xerrors.Unjoin(joinedErr)
	assert.ElementsMatch(t, []error{err1, err2}, result)
}

func TestUnjoinWrappedError(t *testing.T) {
	t.Parallel()

	// Test that Unjoin doesn't unwrap wrapped errors - it only works on joined errors
	baseErr := errors.New("base error")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)

	result := xerrors.Unjoin(wrappedErr)
	assert.Equal(t, []error{wrappedErr}, result) // Should return the wrapped error as-is
}

func TestUnjoinWrappedJoinedError(t *testing.T) {
	t.Parallel()

	// Test that Unjoin doesn't unwrap a wrapped joined error
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joinedErr := errors.Join(err1, err2)
	wrappedJoinedErr := fmt.Errorf("wrapped: %w", joinedErr)

	result := xerrors.Unjoin(wrappedJoinedErr)
	assert.Equal(t, []error{wrappedJoinedErr}, result) // Should return the wrapped joined error as-is
}

func TestUnjoinNestedJoin(t *testing.T) {
	t.Parallel()

	// Test that Unjoin only returns direct children, not grandchildren
	errA := errors.New("error A")
	errB := errors.New("error B")
	errC := errors.New("error C")
	errD := errors.New("error D")

	// Create nested join
	errAB := errors.Join(errA, errB)
	errCD := errors.Join(errC, errD)
	errABCD := errors.Join(errAB, errCD)

	result := xerrors.Unjoin(errABCD)
	assert.Len(t, result, 2)
	assert.ElementsMatch(t, []error{errAB, errCD}, result) // Only direct children
}

func TestFlattenNil(t *testing.T) {
	t.Parallel()

	result := xerrors.Flatten(nil)
	assert.Nil(t, result)
}

func TestFlattenSingleError(t *testing.T) {
	t.Parallel()

	err := errors.New("single error")
	result := xerrors.Flatten(err)
	assert.Equal(t, []error{err}, result)
}

func TestFlattenSimpleJoin(t *testing.T) {
	t.Parallel()

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joinedErr := errors.Join(err1, err2)
	result := xerrors.Flatten(joinedErr)
	assert.ElementsMatch(t, []error{err1, err2}, result)
}

func TestFlattenNestedJoins(t *testing.T) {
	t.Parallel()

	// Create a complex nested structure like what we use in tests
	errA := errors.New("error A")
	errB := errors.New("error B")
	errC := errors.New("error C")
	errD := errors.New("error D")
	errE := errors.New("error E")

	// Create nested joins: Join(Join(A,B), Join(E, Join(C,D)))
	errAB := errors.Join(errA, errB)
	errCD := errors.Join(errC, errD)
	errCDE := errors.Join(errE, errCD)
	errABCDE := errors.Join(errAB, errCDE)

	result := xerrors.Flatten(errABCDE)

	// Should flatten to all 5 individual errors
	assert.Len(t, result, 5)
	assert.ElementsMatch(t, []error{errA, errB, errC, errD, errE}, result)
}

func TestFlattenWrappedJoin(t *testing.T) {
	t.Parallel()

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joinedErr := errors.Join(err1, err2)

	// Wrap the joined error
	wrappedErr := fmt.Errorf("wrapped: %w", joinedErr)

	result := xerrors.Flatten(wrappedErr)

	// Should unwrap and then flatten the joined error
	assert.Len(t, result, 2)
	assert.ElementsMatch(t, []error{err1, err2}, result)
}

func TestFlattenDoubleWrappedJoin(t *testing.T) {
	t.Parallel()

	e1 := errors.New("e1")
	e2 := errors.New("e2")
	j := errors.Join(e1, e2)
	doubleWrapped := fmt.Errorf("b: %w", fmt.Errorf("a: %w", j))
	got := xerrors.Flatten(doubleWrapped)
	assert.ElementsMatch(t, []error{e1, e2}, got)
}

func TestUnjoinVsFlatten(t *testing.T) {
	t.Parallel()

	// Create nested structure to demonstrate the difference
	errA := errors.New("error A")
	errB := errors.New("error B")
	errC := errors.New("error C")
	errD := errors.New("error D")

	// Create nested join
	errAB := errors.Join(errA, errB)
	errCD := errors.Join(errC, errD)
	errABCD := errors.Join(errAB, errCD)

	// Unjoin should return only direct children (errAB, errCD)
	unjoinResult := xerrors.Unjoin(errABCD)
	assert.Len(t, unjoinResult, 2)
	assert.ElementsMatch(t, []error{errAB, errCD}, unjoinResult)

	// Flatten should return all leaf errors (errA, errB, errC, errD)
	flattenResult := xerrors.Flatten(errABCD)
	assert.Len(t, flattenResult, 4)
	assert.ElementsMatch(t, []error{errA, errB, errC, errD}, flattenResult)
}

func TestFlattenMixedWrappedAndJoined(t *testing.T) {
	t.Parallel()

	// Test complex case: wrapped joined errors mixed with regular errors
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	joinedErr := errors.Join(err1, err2)
	wrappedJoinedErr := fmt.Errorf("wrapped: %w", joinedErr)

	// Join a regular error with a wrapped joined error
	finalErr := errors.Join(err3, wrappedJoinedErr)

	result := xerrors.Flatten(finalErr)

	// Should flatten to get all 3 individual errors
	assert.Len(t, result, 3)
	assert.ElementsMatch(t, []error{err1, err2, err3}, result)
}
