package sanitizejson

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{
			input:    "github.com/org/repo.Type",
			expected: "github_com/org/repo_Type",
			name:     "package path with dots and slashes",
		},
		{
			input:    "Type[Param]",
			expected: "Type[Param]",
			name:     "generic type with brackets",
		},
		{
			input:    "simple_key",
			expected: "simple_key",
			name:     "already safe key",
		},
		{
			input:    "github.com/zircuit-labs/zkr-go-common/xerrors.ExtendedError[github.com/zircuit-labs/zkr-go-common/xerrors/errclass.Class]",
			expected: "github_com/zircuit-labs/zkr-go-common/xerrors_ExtendedError[github_com/zircuit-labs/zkr-go-common/xerrors/errclass_Class]",
			name:     "complex type path from real usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Key(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify no problematic characters remain
			assert.NotContains(t, result, ".")
		})
	}
}
