package blocker

import (
	"testing"
)

func TestSuffixWildcardMatcher(t *testing.T) {
	matcher := NewSuffixWildcardMatcher("google.*")

	tests := []struct {
		domain   string
		expected bool
	}{
		{"google.com", true},
		{"google.de", true},
		{"google.es", true},
		{"google.co.uk", true},
		{"www.google.com", true},
		{"mail.google.de", true},
		{"notgoogle.com", false},
		{"google", false},
	}

	for _, tt := range tests {
		result := matcher.Match(tt.domain)
		if result != tt.expected {
			t.Errorf("Match(%q) = %v, want %v", tt.domain, result, tt.expected)
		}
	}
}

func TestDoubleWildcardMatcher(t *testing.T) {
	matcher := NewDoubleWildcardMatcher("*.google.*")

	tests := []struct {
		domain   string
		expected bool
	}{
		{"www.google.com", true},
		{"mail.google.de", true},
		{"api.google.co.uk", true},
		{"google.com", false}, // No subdomain prefix
		{"notgoogle.com", false},
	}

	for _, tt := range tests {
		result := matcher.Match(tt.domain)
		if result != tt.expected {
			t.Errorf("Match(%q) = %v, want %v", tt.domain, result, tt.expected)
		}
	}
}

func TestExactMatcher(t *testing.T) {
	matcher := NewExactMatcher("facebook.com")

	tests := []struct {
		domain   string
		expected bool
	}{
		{"facebook.com", true},
		{"www.facebook.com", true},
		{"m.facebook.com", true},
		{"api.facebook.com", true},
		{"notfacebook.com", false},
		{"facebook.de", false},
	}

	for _, tt := range tests {
		result := matcher.Match(tt.domain)
		if result != tt.expected {
			t.Errorf("Match(%q) = %v, want %v", tt.domain, result, tt.expected)
		}
	}
}

func TestPrefixWildcardMatcher(t *testing.T) {
	matcher := NewPrefixWildcardMatcher("*.example.com")

	tests := []struct {
		domain   string
		expected bool
	}{
		{"sub.example.com", true},
		{"www.example.com", true},
		{"deep.sub.example.com", true},
		{"example.com", false}, // Exact match not included
		{"notexample.com", false},
	}

	for _, tt := range tests {
		result := matcher.Match(tt.domain)
		if result != tt.expected {
			t.Errorf("Match(%q) = %v, want %v", tt.domain, result, tt.expected)
		}
	}
}

func TestCreateMatcher(t *testing.T) {
	tests := []struct {
		pattern      string
		expectedType string
	}{
		{"facebook.com", "*blocker.ExactMatcher"},
		{"*.example.com", "*blocker.PrefixWildcardMatcher"},
		{"google.*", "*blocker.SuffixWildcardMatcher"},
		{"*.google.*", "*blocker.DoubleWildcardMatcher"},
	}

	for _, tt := range tests {
		matcher := CreateMatcher(tt.pattern)
		got := sprintf("%T", matcher)
		if got != tt.expectedType {
			t.Errorf("CreateMatcher(%q) type = %v, want %v", tt.pattern, got, tt.expectedType)
		}
	}
}

func sprintf(format string, a interface{}) string {
	return fmt.Sprintf(format, a)
}

var fmt = struct {
	Sprintf func(string, ...interface{}) string
}{
	Sprintf: func(format string, args ...interface{}) string {
		// Simple type name extraction
		if len(args) > 0 {
			switch args[0].(type) {
			case *ExactMatcher:
				return "*blocker.ExactMatcher"
			case *PrefixWildcardMatcher:
				return "*blocker.PrefixWildcardMatcher"
			case *SuffixWildcardMatcher:
				return "*blocker.SuffixWildcardMatcher"
			case *DoubleWildcardMatcher:
				return "*blocker.DoubleWildcardMatcher"
			}
		}
		return ""
	},
}
