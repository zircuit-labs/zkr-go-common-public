package collections

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		set      Set[int]
		expected string
	}{
		{
			name:     "empty set",
			set:      NewSet[int](),
			expected: `[]`,
		},
		{
			name:     "single element",
			set:      NewSet(42),
			expected: `[42]`,
		},
		{
			name: "multiple elements",
			set:  NewSet(1, 2, 3),
			// Note: order is not guaranteed in sets, so we'll check length and contents separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.set)
			require.NoError(t, err)

			if tt.name != "multiple elements" {
				assert.JSONEq(t, tt.expected, string(data))
			} else {
				// For multiple elements, just verify it's a valid JSON array with correct elements
				var unmarshaled []int
				err := json.Unmarshal(data, &unmarshaled)
				require.NoError(t, err)
				assert.Len(t, unmarshaled, 3)
				assert.Contains(t, unmarshaled, 1)
				assert.Contains(t, unmarshaled, 2)
				assert.Contains(t, unmarshaled, 3)
			}
		})
	}
}

func TestSetUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jsonData string
		expected Set[int]
	}{
		{
			name:     "empty array",
			jsonData: `[]`,
			expected: NewSet[int](),
		},
		{
			name:     "single element",
			jsonData: `[42]`,
			expected: NewSet(42),
		},
		{
			name:     "multiple elements",
			jsonData: `[1, 2, 3]`,
			expected: NewSet(1, 2, 3),
		},
		{
			name:     "duplicate elements",
			jsonData: `[1, 2, 2, 3, 1]`,
			expected: NewSet(1, 2, 3), // Duplicates should be removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result Set[int]
			err := json.Unmarshal([]byte(tt.jsonData), &result)
			require.NoError(t, err)

			assert.ElementsMatch(t, result.Members(), tt.expected.Members())
		})
	}
}

func TestSetJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := NewSet("apple", "banana", "cherry")

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back to Set
	var restored Set[string]
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// Should be equal
	assert.ElementsMatch(t, original.Members(), restored.Members())
}

func TestSetUnmarshalJSONInvalidData(t *testing.T) {
	t.Parallel()

	var s Set[int]
	err := json.Unmarshal([]byte(`"not an array"`), &s)
	assert.Error(t, err)

	err = json.Unmarshal([]byte(`{}`), &s)
	assert.Error(t, err)
}
