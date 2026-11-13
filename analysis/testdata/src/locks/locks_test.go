package locks

import "testing"

// TestLocks sends while mu is locked, in a test file.
func TestLocks(t *testing.T) {
	s := &store{ch: make(chan int, 1)}
	s.mu.Lock()
	s.ch <- 1
	s.mu.Unlock()
}
