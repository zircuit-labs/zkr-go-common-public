package pg

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestKeySortString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		keySort  KeySort
		expected string
	}{
		{
			name:     "basic ascending sort",
			keySort:  KeySort{Key: "name", Sort: SortOrderAscending},
			expected: "name ASC",
		},
		{
			name:     "basic descending sort",
			keySort:  KeySort{Key: "age", Sort: SortOrderDescending},
			expected: "age DESC",
		},
		{
			name:     "complex flag ignored",
			keySort:  KeySort{Key: "height", Sort: SortOrderAscending, Complex: true},
			expected: "height ASC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.keySort.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestKeySortOpposite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		keySort  KeySort
		expected string
	}{
		{
			name:     "ascending to descending",
			keySort:  KeySort{Key: "name", Sort: SortOrderAscending},
			expected: "name DESC",
		},
		{
			name:     "descending to ascending",
			keySort:  KeySort{Key: "age", Sort: SortOrderDescending},
			expected: "age ASC",
		},
		{
			name:     "complex flag ignored in opposite",
			keySort:  KeySort{Key: "salary", Sort: SortOrderAscending, Complex: true},
			expected: "salary DESC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.keySort.Opposite()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPaginationSort(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	mockBun := bun.NewDB(db, pgdialect.New())

	mockQuery := mockBun.NewSelect()
	finalQuery := paginationSort[MockData, MockDataOrdered](mockQuery)

	expected := `SELECT * ORDER BY "name" ASC, "name2" DESC, CASE WHEN name = '' THEN 0 ELSE 1 END ASC`

	if finalQuery.String() != expected {
		t.Errorf("expected %q, got %q", expected, finalQuery)
	}
}

func TestPaginationReverseSort(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	mockBun := bun.NewDB(db, pgdialect.New())

	mockQuery := mockBun.NewSelect()
	finalQuery := paginationReverseSort[MockData, MockDataOrdered](mockQuery)

	expected := `SELECT * ORDER BY "name" DESC, "name2" ASC, CASE WHEN name = '' THEN 0 ELSE 1 END DESC`

	if finalQuery.String() != expected {
		t.Errorf("expected %q, got %q", expected, finalQuery)
	}
}

type (
	MockData struct {
		bun.BaseModel `bun:"table:mock_data"`
	}
	MockDataOrdered struct{}
)

func (c MockDataOrdered) KeySort() []KeySort {
	return []KeySort{
		{Key: "name", Sort: SortOrderAscending},
		{Key: "name2", Sort: SortOrderDescending},
		{Key: "CASE WHEN name = '' THEN 0 ELSE 1 END", Sort: SortOrderAscending, Complex: true},
	}
}

func (c MockDataOrdered) CursorValues() []string {
	return nil
}

func (c MockDataOrdered) DeserizalizeCursorValues(values []string) ([]any, error) {
	return nil, nil
}

func (c MockDataOrdered) UnWrap() MockData {
	return MockData{}
}
