package logs

import (
	"sync"
	"time"
)

// Entry is a single request log line or lifecycle message.
type Entry struct {
	Time    time.Time
	Message string
	Method  string
	Path    string
	Status  int
	Latency time.Duration
}

// Store is a bounded in-memory ring buffer of log entries.
type Store struct {
	mu      sync.Mutex
	entries []Entry
	max     int
}

// New creates a store that retains at most max entries.
func New(max int) *Store {
	return &Store{max: max}
}

// Add appends an entry, dropping the oldest once the buffer is full.
func (s *Store) Add(e Entry) {
	if s.max <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	if len(s.entries) > s.max {
		s.entries = s.entries[len(s.entries)-s.max:]
	}
}

// AddEvent records a lifecycle message (e.g. proxy started/stopped).
func (s *Store) AddEvent(msg string) {
	s.Add(Entry{Time: time.Now(), Message: msg})
}

// Entries returns a snapshot of the retained entries in order.
func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}
