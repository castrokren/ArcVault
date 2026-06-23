package runner

import (
	"testing"
)

// TestParseCommandArgs tests the command parsing logic that prevents shell injection.
func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "simple command",
			input:  "echo hello",
			expect: []string{"echo", "hello"},
		},
		{
			name:   "single argument",
			input:  "echo",
			expect: []string{"echo"},
		},
		{
			name:   "multiple arguments",
			input:  "rsync -a --info=progress2 /src /dst",
			expect: []string{"rsync", "-a", "--info=progress2", "/src", "/dst"},
		},
		{
			name:   "quoted argument with spaces",
			input:  `echo "hello world"`,
			expect: []string{"echo", "hello world"},
		},
		{
			name:   "single quoted argument",
			input:  "echo 'hello world'",
			expect: []string{"echo", "hello world"},
		},
		{
			name:   "mixed quotes",
			input:  "robocopy \"C:\\Source\" 'D:\\Dest' /E /NFL",
			expect: []string{"robocopy", "C:\\Source", "D:\\Dest", "/E", "/NFL"},
		},
		{
			name:   "empty string",
			input:  "",
			expect: []string{},
		},
		{
			name:   "only whitespace",
			input:  "   \t   ",
			expect: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommandArgs(tt.input)
			if len(result) != len(tt.expect) {
				t.Errorf("expected %d args, got %d: %v", len(tt.expect), len(result), result)
				return
			}
			for i := range result {
				if result[i] != tt.expect[i] {
					t.Errorf("arg %d: expected %q, got %q", i, tt.expect[i], result[i])
				}
			}
		})
	}
}
