package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun"

	"github.com/zircuit-labs/zkr-go-common/xerrors/stacktrace"
)

var ErrCursorValues = errors.New("unable to deserialize expected cursor values")

type SortOrder string

type Cursor struct {
	Next     string
	Previous string
}

func (c Cursor) IsReverse() bool {
	return c.Previous != ""
}

func (c Cursor) Exists() bool {
	return c.Next != "" || c.Previous != ""
}

const (
	SortOrderAscending  SortOrder = "ASC"
	SortOrderDescending SortOrder = "DESC"
)

func (so SortOrder) Opposite() SortOrder {
	if so == SortOrderAscending {
		return SortOrderDescending
	}
	return SortOrderAscending
}

type ComparisionOperator string

const (
	ComparisionOperatorGreaterThan ComparisionOperator = ">"
	ComparisionOperatorLessThan    ComparisionOperator = "<"
	ComparisionOperatorEqual       ComparisionOperator = "="
)

func (co ComparisionOperator) Opposite() ComparisionOperator {
	switch co {
	case ComparisionOperatorGreaterThan:
		return ComparisionOperatorLessThan
	case ComparisionOperatorLessThan:
		return ComparisionOperatorGreaterThan
	default:
		return ComparisionOperatorEqual
	}
}

type KeySort struct {
	Key     string
	Sort    SortOrder
	Complex bool
}

func (k KeySort) String() string {
	return fmt.Sprintf("%s %s", k.Key, k.Sort)
}

func (k KeySort) Opposite() string {
	return fmt.Sprintf("%s %s", k.Key, k.Sort.Opposite())
}

type QueryOpts interface {
	GetLimit() int
	GetCursor() Cursor
}

// Pagable defines how cursor pagination should be implemented for a given struct.
type Pageable[V any] interface {
	KeySort() []KeySort                                      // eg [{"l2_block_index", SortOrderDescending}, {}"tx_index", SortOrderDescending}]
	CursorValues() []string                                  // values provided as strings in the same order as KeySort eg ["123", 456]
	DeserizalizeCursorValues(values []string) ([]any, error) // convert cursor values to their respective types
	UnWrap() V                                               // return the underlying struct
}

func Paginate[V any, T Pageable[V]](ctx context.Context, filterQuery *bun.SelectQuery, opts QueryOpts) (results []*V, cursor Cursor, err error) {
	var data []T

	// If no cursor is present, start from the beginning
	if !opts.GetCursor().Exists() {
		filterQuery = paginationSort[V, T](filterQuery)
		if opts.GetLimit() > 0 {
			filterQuery = filterQuery.Limit(opts.GetLimit() + 1) // Fetch one more than the limit to determine if there are more results
		}

		// Execute the query
		err := filterQuery.Scan(ctx, &data)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, cursor, stacktrace.Wrap(err)
		}
		if errors.Is(err, sql.ErrNoRows) || len(data) == 0 {
			return nil, cursor, nil
		}

		// If no limit is set, return all results without pagination.
		if opts.GetLimit() == 0 {
			return parseOrderedWrapper(data), cursor, nil
		}

		// If the number of results is greater than the limit, there are more results to fetch.
		moreData := (len(data) > opts.GetLimit())
		if moreData {
			data = data[:len(data)-1]
		}

		// Return cursor values for later use.
		if moreData {
			cursor.Next = strings.Join(data[len(data)-1].CursorValues(), ",")
		}
		// cursor.Previous should remain empty as this is the first page of results.

		return parseOrderedWrapper(data), cursor, nil
	}

	// Otherwise, apply cursor style pagination
	filterQuery, err = paginationWhere[V, T](filterQuery, opts.GetCursor())
	if err != nil {
		return nil, cursor, stacktrace.Wrap(err)
	}
	if opts.GetLimit() > 0 {
		filterQuery = filterQuery.Limit(opts.GetLimit() + 1) // Fetch one more than the limit to determine if there are more results
	}

	// Execute the query
	err = filterQuery.Scan(ctx, &data)
	if errors.Is(err, sql.ErrNoRows) || len(data) == 0 {
		return nil, cursor, nil
	}
	if err != nil {
		return nil, cursor, stacktrace.Wrap(err)
	}

	// If no limit is set, return all results without pagination
	if opts.GetLimit() == 0 {
		return parseOrderedWrapper(data), cursor, nil
	}

	// If the number of results is greater than the limit, there are more results to fetch.
	moreData := (len(data) > opts.GetLimit())

	// Reverse the returned data to the proper order if executing a reverse cursor query
	// NOTE: Cursors should be empty in the search direction if no more data is available.
	if opts.GetCursor().IsReverse() {
		if moreData {
			data = data[1:]
		}
		slices.Reverse(data)
		if moreData {
			cursor.Previous = strings.Join(data[0].CursorValues(), ",")
		}
		cursor.Next = strings.Join(data[len(data)-1].CursorValues(), ",")
	} else {
		if moreData {
			data = data[:len(data)-1]
			cursor.Next = strings.Join(data[len(data)-1].CursorValues(), ",")
		}
		cursor.Previous = strings.Join(data[0].CursorValues(), ",")
	}

	return parseOrderedWrapper(data), cursor, nil
}

func paginationSort[V any, T Pageable[V]](q *bun.SelectQuery) *bun.SelectQuery {
	var data T
	for _, keySort := range data.KeySort() {
		if keySort.Complex {
			q.OrderExpr(keySort.String())
			continue
		}

		q.Order(keySort.String())
	}
	return q
}

func paginationReverseSort[V any, T Pageable[V]](q *bun.SelectQuery) *bun.SelectQuery {
	var data T
	for _, keySort := range data.KeySort() {
		if keySort.Complex {
			q.OrderExpr(keySort.Opposite())
			continue
		}
		q.Order(keySort.Opposite())
	}
	return q
}

type clause struct {
	key        string
	sort       SortOrder
	comparator ComparisionOperator
	value      any
}

func (cl clause) String() string {
	return fmt.Sprintf("%s %s ?", cl.key, cl.comparator)
}

func (cl clause) EqualityString() string {
	return fmt.Sprintf("%s %s ?", cl.key, ComparisionOperatorEqual)
}

func paginationWhere[V any, T Pageable[V]](q *bun.SelectQuery, cur Cursor) (*bun.SelectQuery, error) {
	var data T

	// Deserialize the cursor values
	cursorValue := cur.Next
	if cur.Previous != "" {
		cursorValue = cur.Previous
	}
	cursorValues := strings.Split(cursorValue, ",")
	actualCursorValues, err := data.DeserizalizeCursorValues(cursorValues)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}

	if len(actualCursorValues) != len(data.KeySort()) {
		return nil, stacktrace.Wrap(ErrCursorValues)
	}

	// Build the where clause(s)
	clauses := make([]clause, 0, len(data.KeySort()))
	for i, keySort := range data.KeySort() {
		cl := clause{
			key:   keySort.Key,
			sort:  keySort.Sort,
			value: actualCursorValues[i],
		}

		if keySort.Sort == SortOrderAscending {
			cl.comparator = ComparisionOperatorGreaterThan
		} else {
			cl.comparator = ComparisionOperatorLessThan
		}

		// For previous page, we must reversing the sort order and the comparator
		// NOTE: The query results will need to be reversed before being returned.
		if cur.Previous != "" {
			cl.comparator = cl.comparator.Opposite()
			cl.sort = cl.sort.Opposite()
		}
		clauses = append(clauses, cl)
	}

	// Each clause now needs to consider equality of the previous clauses, then OR'd together.
	// For example, with two keys, the final where clause might look like:
	// `WHERE (key1 > ?) or (key1 = ? and key2 > ?)`
	fullClauses := make([]string, 0, len(clauses))
	numValues := (len(clauses) * (len(clauses) + 1)) / 2 // sum of 1 to n
	valueSet := make([]any, 0, numValues)

	for i, clause := range clauses {
		subClauses := make([]string, 0, i)
		for _, previousClause := range clauses[:i] {
			subClauses = append(subClauses, previousClause.EqualityString())
			valueSet = append(valueSet, previousClause.value)
		}
		subClauses = append(subClauses, clause.String())
		valueSet = append(valueSet, clause.value)
		fullClauses = append(fullClauses, fmt.Sprintf("(%s)", strings.Join(subClauses, " AND ")))
	}
	compoundWhereClause := strings.Join(fullClauses, " OR ")

	// Apply the where clause
	filterQuery := q.Where(compoundWhereClause, valueSet...)

	// Include the appropriate sort order
	if cur.Next != "" {
		filterQuery = paginationSort[V, T](filterQuery)
	} else {
		filterQuery = paginationReverseSort[V, T](filterQuery)
	}
	return filterQuery, nil
}

func parseOrderedWrapper[V any, T Pageable[V]](ordered []T) []*V {
	parsed := make([]*V, 0, len(ordered))
	for _, t := range ordered {
		value := t.UnWrap()
		parsed = append(parsed, &value)
	}
	return parsed
}
