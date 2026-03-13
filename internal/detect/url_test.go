package detect

import "testing"

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com", true},
		{"http://x.co/path?q=1&r=2", true},
		{"ftp://files.example.com/file.txt", true},
		{"http://localhost:8080", true},
		{"not a url", false},
		{"", false},
		{"example.com", false},          // no scheme
		{"javascript:alert(1)", false},  // disallowed scheme
		{"mailto:user@example.com", false}, // disallowed scheme
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsURL(tt.input)
			if got != tt.want {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
