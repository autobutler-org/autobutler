package stringutil

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"double digit", 99, "99"},
		{"three digits", 999, "999"},
		{"thousand", 1000, "1,000"},
		{"thousand and one", 1001, "1,001"},
		{"ten thousand", 10000, "10,000"},
		{"hundred thousand", 100000, "100,000"},
		{"million", 1000000, "1,000,000"},
		{"arbitrary large", 1234567, "1,234,567"},
		{"max test value", 999999999, "999,999,999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrimLeading(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		char     rune
		expected string
	}{
		{"no leading chars", "hello", '/', "hello"},
		{"single leading slash", "/path", '/', "path"},
		{"multiple leading slashes", "///path", '/', "path"},
		{"no match", "path", '/', "path"},
		{"empty string", "", '/', ""},
		{"all chars", "///", '/', ""},
		{"leading zeros", "000123", '0', "123"},
		{"leading spaces", "   text", ' ', "text"},
		{"mixed but trim one char", "///path/to/file", '/', "path/to/file"},
		{"single char string to trim", "/", '/', ""},
		{"char in middle only", "path/to/file", '/', "path/to/file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimLeading(tt.input, tt.char)
			if result != tt.expected {
				t.Errorf("TrimLeading(%q, %q) = %q; want %q", tt.input, tt.char, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber_EdgeCases(t *testing.T) {
	// Test edge case with negative numbers (though function expects positive)
	// This tests the actual behavior
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"1234", 1234, "1,234"},
		{"12345", 12345, "12,345"},
		{"123456", 123456, "123,456"},
		{"1234567890", 1234567890, "1,234,567,890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrimLeading_DifferentRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		char     rune
		expected string
	}{
		{"trim a", "aaabcd", 'a', "bcd"},
		{"trim x", "xxxyyz", 'x', "yyz"},
		{"trim dash", "---test", '-', "test"},
		{"trim underscore", "___var", '_', "var"},
		{"trim dot", "...file", '.', "file"},
		{"unicode char", "✓✓✓text", '✓', "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimLeading(tt.input, tt.char)
			if result != tt.expected {
				t.Errorf("TrimLeading(%q, %q) = %q; want %q", tt.input, tt.char, result, tt.expected)
			}
		})
	}
}
