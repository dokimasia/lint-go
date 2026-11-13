// Package deps is a dependency of the package uses, whose functions lockblock summarizes in facts.
package deps

import (
	"sync"
	"time"
)

// Pause sleeps for a millisecond.
func Pause() { // want Pause:"time.Sleep"
	time.Sleep(time.Millisecond)
}

// Store has a method that pauses.
type Store struct{}

// Flush pauses.
func (*Store) Flush() { // want Flush:"deps.Pause -> time.Sleep"
	Pause()
}

// Quiet makes no blocking operation.
func Quiet() int {
	return 1
}

// Await receives from ch while mu, the lock of its caller, is unlocked, and then locks mu again.
func Await(mu *sync.Mutex, ch chan int) { // want Await:"^a receive from a channel when its callers hold 2 locks$"
	mu.Unlock()
	<-ch
	mu.Lock()
}
