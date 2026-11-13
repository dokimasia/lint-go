// Package uses is the package of the cases of lockblock through the calls of another package.
package uses

import (
	"sync"

	"deps"
)

// flush calls the functions of deps while mu is locked.
func flush(mu *sync.Mutex, s *deps.Store, ch chan int) { // want flush:"deps.Pause -> time.Sleep"
	mu.Lock()
	deps.Pause() // want `the call of deps.Pause blocks inside the critical section of mu from line 12: deps.Pause -> time.Sleep`
	s.Flush()    // want `the call of \(\*deps.Store\).Flush blocks inside the critical section of mu from line 12: \(\*deps.Store\).Flush -> deps.Pause -> time.Sleep`
	deps.Quiet()
	deps.Await(mu, ch)
	mu.Unlock()
}

// await calls deps.Await while other and mu are locked.
func await(mu, other *sync.Mutex, ch chan int) { // want await:"^deps.Await -> a receive from a channel$"
	other.Lock()
	mu.Lock()
	deps.Await(mu, ch) // want `the call of deps.Await blocks inside the critical section of other from line 22: deps.Await -> a receive from a channel`
	mu.Unlock()
	other.Unlock()
}
