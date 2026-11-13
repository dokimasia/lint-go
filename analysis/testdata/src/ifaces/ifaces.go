// Package ifaces is the package of the cases of lockblock with interfaces that the settings name.
package ifaces

import "sync"

// Device is an interface whose methods block, as the setting interfaces names it.
type Device interface {
	Sync() error
}

// Plugin is an interface whose methods run code outside the module, as the setting hooks names it.
type Plugin interface {
	Run()
}

// Volume embeds Device, and has a method of its own.
type Volume interface {
	Device
	Size() int
}

// calls calls each method while mu is locked.
func calls(mu *sync.Mutex, d Device, p Plugin, v Volume) { // want calls:"ifaces.Device.Sync of a blocking interface"
	mu.Lock()
	_ = d.Sync() // want `ifaces.Device.Sync of a blocking interface blocks inside the critical section of mu from line 24`
	p.Run()      // want `the hook ifaces.Plugin.Run blocks inside the critical section of mu from line 24`
	_ = v.Sync() // want `ifaces.Device.Sync of a blocking interface blocks inside the critical section of mu from line 24`
	_ = v.Size()
	mu.Unlock()
}

// syncs calls Sync of d.
func syncs(d Device) { // want syncs:"ifaces.Device.Sync of a blocking interface"
	_ = d.Sync()
}

// through calls syncs while mu is locked.
func through(mu *sync.Mutex, d Device) { // want through:"ifaces.syncs -> ifaces.Device.Sync of a blocking interface"
	mu.Lock()
	syncs(d) // want `the call of ifaces.syncs blocks inside the critical section of mu from line 39: ifaces.syncs -> ifaces.Device.Sync of a blocking interface`
	mu.Unlock()
}
