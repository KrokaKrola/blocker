package blocker

import (
	"strings"
)

// Matcher defines the interface for domain matching
type Matcher interface {
	Match(domain string) bool
	Pattern() string
}

// ExactMatcher matches exact domain and all its subdomains
type ExactMatcher struct {
	domain string
}

// NewExactMatcher creates a new exact domain matcher
func NewExactMatcher(domain string) *ExactMatcher {
	return &ExactMatcher{
		domain: strings.ToLower(strings.TrimSpace(domain)),
	}
}

// Match checks if the given domain matches
// It matches the exact domain and any subdomain
func (m *ExactMatcher) Match(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Exact match
	if domain == m.domain {
		return true
	}

	// Subdomain match (e.g., "www.facebook.com" matches "facebook.com")
	if strings.HasSuffix(domain, "."+m.domain) {
		return true
	}

	return false
}

// Pattern returns the original pattern
func (m *ExactMatcher) Pattern() string {
	return m.domain
}

// PrefixWildcardMatcher matches domains with prefix wildcard patterns like "*.example.com"
type PrefixWildcardMatcher struct {
	pattern string
	suffix  string
}

// NewPrefixWildcardMatcher creates a new prefix wildcard matcher
// Supports patterns like "*.example.com"
func NewPrefixWildcardMatcher(pattern string) *PrefixWildcardMatcher {
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Extract suffix from pattern like "*.example.com"
	suffix := ""
	if strings.HasPrefix(pattern, "*.") {
		suffix = pattern[1:] // Keep the dot: ".example.com"
	}

	return &PrefixWildcardMatcher{
		pattern: pattern,
		suffix:  suffix,
	}
}

// Match checks if the given domain matches the wildcard pattern
func (m *PrefixWildcardMatcher) Match(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	if m.suffix == "" {
		return false
	}

	// Check if domain ends with the suffix
	// "*.example.com" matches "sub.example.com" but not "example.com"
	return strings.HasSuffix(domain, m.suffix) && domain != m.suffix[1:]
}

// Pattern returns the original pattern
func (m *PrefixWildcardMatcher) Pattern() string {
	return m.pattern
}

// SuffixWildcardMatcher matches domains with suffix wildcard patterns like "google.*"
type SuffixWildcardMatcher struct {
	pattern string
	prefix  string // e.g., "google." for pattern "google.*"
}

// NewSuffixWildcardMatcher creates a new suffix wildcard matcher
// Supports patterns like "google.*" to match google.com, google.de, google.es, etc.
func NewSuffixWildcardMatcher(pattern string) *SuffixWildcardMatcher {
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Extract prefix from pattern like "google.*"
	prefix := ""
	if strings.HasSuffix(pattern, ".*") {
		prefix = pattern[:len(pattern)-1] // Keep the dot: "google."
	}

	return &SuffixWildcardMatcher{
		pattern: pattern,
		prefix:  prefix,
	}
}

// Match checks if the given domain matches the suffix wildcard pattern
func (m *SuffixWildcardMatcher) Match(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	if m.prefix == "" {
		return false
	}

	// "google.*" matches "google.com", "google.de", "www.google.com", etc.
	// Check if domain starts with prefix OR contains ".prefix"
	if strings.HasPrefix(domain, m.prefix) {
		// Make sure there's something after the prefix (the TLD)
		rest := domain[len(m.prefix):]
		// Should not contain another dot (e.g., "google.co.uk" - rest would be "co.uk")
		// Actually, we should allow this, so just check it's not empty
		return len(rest) > 0
	}

	// Also match subdomains like "www.google.com" for pattern "google.*"
	if strings.Contains(domain, "."+m.prefix) {
		return true
	}

	return false
}

// Pattern returns the original pattern
func (m *SuffixWildcardMatcher) Pattern() string {
	return m.pattern
}

// DoubleWildcardMatcher matches domains with wildcards on both sides like "*.google.*"
type DoubleWildcardMatcher struct {
	pattern string
	middle  string // e.g., ".google." for pattern "*.google.*"
}

// NewDoubleWildcardMatcher creates a new double wildcard matcher
// Supports patterns like "*.google.*" to match www.google.com, mail.google.de, etc.
func NewDoubleWildcardMatcher(pattern string) *DoubleWildcardMatcher {
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Extract middle from pattern like "*.google.*"
	middle := ""
	if strings.HasPrefix(pattern, "*.") && strings.HasSuffix(pattern, ".*") {
		middle = pattern[1 : len(pattern)-1] // ".google."
	}

	return &DoubleWildcardMatcher{
		pattern: pattern,
		middle:  middle,
	}
}

// Match checks if the given domain matches the double wildcard pattern
func (m *DoubleWildcardMatcher) Match(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	if m.middle == "" {
		return false
	}

	// "*.google.*" matches "www.google.com", "mail.google.de", etc.
	return strings.Contains(domain, m.middle)
}

// Pattern returns the original pattern
func (m *DoubleWildcardMatcher) Pattern() string {
	return m.pattern
}

// CreateMatcher creates the appropriate matcher for a pattern
func CreateMatcher(pattern string) Matcher {
	pattern = strings.TrimSpace(pattern)

	hasPrefix := strings.HasPrefix(pattern, "*.")
	hasSuffix := strings.HasSuffix(pattern, ".*")

	// Double wildcard: *.google.*
	if hasPrefix && hasSuffix {
		return NewDoubleWildcardMatcher(pattern)
	}

	// Prefix wildcard: *.example.com
	if hasPrefix {
		return NewPrefixWildcardMatcher(pattern)
	}

	// Suffix wildcard: google.*
	if hasSuffix {
		return NewSuffixWildcardMatcher(pattern)
	}

	// Exact match (with automatic subdomain matching)
	return NewExactMatcher(pattern)
}
