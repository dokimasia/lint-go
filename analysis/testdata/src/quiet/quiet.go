// Package quiet sends while a lock is held, and the cases of lockblock skip it or name no interface of it.
package quiet

import "sync"

// Device is an interface that no setting of the cases names.
type Device interface {
	Sync() error
}

// send sends while mu is locked.
func send(mu *sync.Mutex, ch chan int) { // want send:"a send to a channel"
	mu.Lock()
	ch <- 1
	mu.Unlock()
}

// calls calls Sync of d while mu is locked.
func calls(mu *sync.Mutex, d Device) { // want calls:"quiet.Device.Sync of a blocking interface"
	mu.Lock()
	_ = d.Sync()
	mu.Unlock()
}
