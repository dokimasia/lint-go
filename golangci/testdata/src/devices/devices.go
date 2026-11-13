// Package devices calls the interfaces that the settings of the cases of the plugin lockblock name.
package devices

import "sync"

// Device is an interface whose methods block.
type Device interface {
	Sync() error
}

// Plugin is an interface whose methods run code outside the module.
type Plugin interface {
	Run()
}

// calls calls each method while mu is locked.
func calls(mu *sync.Mutex, d Device, p Plugin) { // want calls:"devices.Device.Sync of a blocking interface"
	mu.Lock()
	_ = d.Sync() // want `devices.Device.Sync of a blocking interface blocks inside the critical section of mu from line 18`
	p.Run()      // want `the hook devices.Plugin.Run blocks inside the critical section of mu from line 18`
	mu.Unlock()
}
