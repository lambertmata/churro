package utils

import (
	"testing"
)

func TestSplitString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter string
		expected  []string
	}{
		{
			name:      "basic split",
			input:     "a,b,c",
			delimiter: ",",
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "split with empty parts",
			input:     "a,,b,c",
			delimiter: ",",
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "split with leading/trailing delimiters",
			input:     ",a,b,c,",
			delimiter: ",",
			expected:  []string{"a", "b", "c"},
		},
		{
			name:      "empty string",
			input:     "",
			delimiter: ",",
			expected:  []string{},
		},
		{
			name:      "only delimiters",
			input:     ",,,,",
			delimiter: ",",
			expected:  []string{},
		},
		{
			name:      "no delimiters",
			input:     "abc",
			delimiter: ",",
			expected:  []string{"abc"},
		},
		{
			name:      "pipe delimiter",
			input:     "required|min:5|max:10",
			delimiter: "|",
			expected:  []string{"required", "min:5", "max:10"},
		},
		{
			name:      "colon delimiter",
			input:     "min:5:extra",
			delimiter: ":",
			expected:  []string{"min", "5", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitString(tt.input, tt.delimiter)

			if len(result) != len(tt.expected) {
				t.Errorf("SplitString() length = %d, expected %d", len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("SplitString()[%d] = %v, expected %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func BenchmarkSplitString(b *testing.B) {
	input := "required|min:5|max:100|email|uuid"
	delimiter := "|"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SplitString(input, delimiter)
	}
}
