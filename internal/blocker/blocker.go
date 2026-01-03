package blocker

import (
	"log"
	"strings"
	"sync"
)

// Blocker manages the blacklist and checks domains
type Blocker struct {
	matchers   []Matcher
	mu         sync.RWMutex
	logBlocked bool
	logAllowed bool
	
	// Statistics
	blockedCount int64
	allowedCount int64
	statsMu      sync.Mutex
}

// New creates a new Blocker instance
func New() *Blocker {
	return &Blocker{
		matchers:   make([]Matcher, 0),
		logBlocked: true,
		logAllowed: false,
	}
}

// SetLogging configures logging behavior
func (b *Blocker) SetLogging(logBlocked, logAllowed bool) {
	b.logBlocked = logBlocked
	b.logAllowed = logAllowed
}

// UpdateBlacklist replaces the current blacklist with new patterns
func (b *Blocker) UpdateBlacklist(patterns []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.matchers = make([]Matcher, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		b.matchers = append(b.matchers, CreateMatcher(pattern))
	}
	
	log.Printf("[blocker] Updated blacklist with %d patterns", len(b.matchers))
}

// IsBlocked checks if a domain should be blocked
func (b *Blocker) IsBlocked(domain string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Extract domain from host:port if needed
	if idx := strings.LastIndex(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	for _, matcher := range b.matchers {
		if matcher.Match(domain) {
			b.recordBlocked()
			if b.logBlocked {
				log.Printf("[BLOCKED] %s (matched: %s)", domain, matcher.Pattern())
			}
			return true
		}
	}
	
	b.recordAllowed()
	if b.logAllowed {
		log.Printf("[ALLOWED] %s", domain)
	}
	return false
}

// recordBlocked increments the blocked counter
func (b *Blocker) recordBlocked() {
	b.statsMu.Lock()
	b.blockedCount++
	b.statsMu.Unlock()
}

// recordAllowed increments the allowed counter
func (b *Blocker) recordAllowed() {
	b.statsMu.Lock()
	b.allowedCount++
	b.statsMu.Unlock()
}

// Stats returns current statistics
func (b *Blocker) Stats() (blocked, allowed int64) {
	b.statsMu.Lock()
	defer b.statsMu.Unlock()
	return b.blockedCount, b.allowedCount
}

// GetPatterns returns current blacklist patterns
func (b *Blocker) GetPatterns() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	patterns := make([]string, len(b.matchers))
	for i, m := range b.matchers {
		patterns[i] = m.Pattern()
	}
	return patterns
}
