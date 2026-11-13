// Package unnamed calls an interface that no setting of the cases of lockblock names.
package unnamed

import "sync"

// Device is an interface that no setting of the cases names.
type Device interface {
	Sync() error
}

// calls sends and calls Sync of d while mu is locked.
func calls(mu *sync.Mutex, d Device, ch chan int) { // want calls:"a send to a channel"
	mu.Lock()
	ch <- 1 // want `a send to a channel blocks inside the critical section of mu from line 13`
	_ = d.Sync()
	mu.Unlock()
}
