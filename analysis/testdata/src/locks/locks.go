// Package locks is the package of the cases of lockblock inside one function.
package locks

import (
	"sync"
	"time"
)

// store has the locks, the channel, the group and the cond of the cases.
type store struct {
	sync.Mutex
	mu   sync.Mutex
	rw   sync.RWMutex
	ch   chan int
	wg   sync.WaitGroup
	cond *sync.Cond
}

// send sends while mu is locked.
func (s *store) send() { // want send:"a send to a channel"
	s.mu.Lock()
	s.ch <- 1 // want `a send to a channel blocks inside the critical section of s.mu from line 21`
	s.mu.Unlock()
}

// receive receives while mu is locked until the function returns.
func (s *store) receive() int { // want receive:"a receive from a channel"
	s.mu.Lock()
	defer s.mu.Unlock()
	return <-s.ch // want `a receive from a channel blocks inside the critical section of s.mu from line 28`
}

// choose selects without a default case while mu is locked.
func (s *store) choose(other chan int) { // want choose:"a select without a default case"
	s.mu.Lock()
	select { // want `a select without a default case blocks inside the critical section of s.mu from line 35`
	case <-s.ch:
	case <-other:
	}
	s.mu.Unlock()
}

// poll selects with a default case while mu is locked, which does not block.
func (s *store) poll(other chan int) {
	s.mu.Lock()
	select {
	case <-s.ch:
	case <-other:
	default:
	}
	s.mu.Unlock()
}

// sleep sleeps and waits for the group while the reader lock of rw is held.
func (s *store) sleep() { // want sleep:"time.Sleep"
	s.rw.RLock()
	time.Sleep(time.Millisecond) // want `time.Sleep blocks inside the critical section of s.rw from line 56`
	s.wg.Wait()                  // want `sync.WaitGroup.Wait blocks inside the critical section of s.rw from line 56`
	s.rw.RUnlock()
}

// embedded sends while the embedded lock is held.
func (s *store) embedded() { // want embedded:"a send to a channel"
	s.Lock()
	s.ch <- 1 // want `a send to a channel blocks inside the critical section of s from line 64`
	s.Unlock()
}

// wait waits on the cond of mu while mu alone is locked, which releases mu.
func (s *store) wait() { // want wait:"^sync.Cond.Wait$"
	s.mu.Lock()
	s.cond.Wait()
	s.mu.Unlock()
}

// waitUnderTwo waits on the cond of mu while rw and mu are locked, so rw stays locked.
func (s *store) waitUnderTwo() { // want waitUnderTwo:"^sync.Cond.Wait$"
	s.rw.Lock()
	s.mu.Lock()
	s.cond.Wait() // want `sync.Cond.Wait blocks inside the critical section of s.rw from line 78`
	s.mu.Unlock()
	s.rw.Unlock()
}

// released sends once mu is unlocked again.
func (s *store) released() { // want released:"^a send to a channel$"
	s.mu.Lock()
	s.mu.Unlock()
	s.ch <- 1
}

// onePath sends where one path alone holds mu.
func (s *store) onePath(lock bool) { // want onePath:"a send to a channel"
	if lock {
		s.mu.Lock()
	}
	s.ch <- 1
	if lock {
		s.mu.Unlock()
	}
}

// tried sends after TryLock, which begins no critical section.
func (s *store) tried() { // want tried:"a send to a channel"
	if s.mu.TryLock() {
		s.ch <- 1
		s.mu.Unlock()
	}
}

// local sends while a local lock is held.
func local(ch chan int) { // want local:"a send to a channel"
	var mu sync.Mutex
	mu.Lock()
	ch <- 1 // want `a send to a channel blocks inside the critical section of mu from line 114`
	mu.Unlock()
}

// spawned sends in a goroutine and in a deferred call while mu is locked, which do not block it.
func (s *store) spawned() {
	s.mu.Lock()
	defer s.mu.Unlock()
	go func() { s.ch <- 1 }()
	defer func() { s.ch <- 2 }()
}

// shards has an array of locks, a pointer to a lock and a channel.
type shards struct {
	locks [2]sync.Mutex
	mup   *sync.Mutex
	ch    chan int
}

// indexed sends while a lock of the array is held.
func (s *shards) indexed(i int) { // want indexed:"a send to a channel"
	s.locks[i].Lock()
	s.ch <- 1 // want `a send to a channel blocks inside the critical section of s.locks\[i\] from line 136`
	s.locks[i].Unlock()
}

// pointed sends while the lock that mup points to is held.
func (s *shards) pointed() { // want pointed:"a send to a channel"
	s.mup.Lock()
	s.ch <- 1 // want `a send to a channel blocks inside the critical section of s.mup from line 143`
	s.mup.Unlock()
}

// expressed sends while a lock is held that a method expression locks.
func expressed(mu *sync.Mutex, ch chan int) { // want expressed:"a send to a channel"
	(*sync.Mutex).Lock(mu)
	ch <- 1 // want `a send to a channel blocks inside the critical section of mu from line 150`
	mu.Unlock()
}

// bound sends while a lock is held that a method value locks, which the analyzer does not track.
func bound(mu *sync.Mutex, ch chan int) { // want bound:"a send to a channel"
	lock := mu.Lock
	lock()
	ch <- 1
	mu.Unlock()
}

// waitAlone waits on the cond of the lock of its caller, which releases that lock.
func (s *store) waitAlone() { // want waitAlone:"^sync.Cond.Wait when its callers hold 2 locks$"
	s.cond.Wait()
}

// handOff receives while the lock mu of its caller is unlocked, and then locks mu again.
func (s *store) handOff() { // want handOff:"^a receive from a channel when its callers hold 2 locks$"
	s.mu.Unlock()
	<-s.ch
	s.mu.Lock()
}

// repeated receives n times while the lock mu of its caller is unlocked.
func (s *store) repeated(n int) { // want repeated:"^a receive from a channel when its callers hold 2 locks$"
	for range n {
		s.mu.Unlock()
		<-s.ch
		s.mu.Lock()
	}
}

// maybe receives while the lock mu of its caller is unlocked when wait is true, and then sends
// while mu is locked again.
func (s *store) maybe(wait bool) { // want maybe:"^a send to a channel$"
	if wait {
		s.mu.Unlock()
		<-s.ch
		s.mu.Lock()
	}
	s.ch <- 1
}

// swapped sends while rw is locked and the lock mu of its caller is unlocked.
func (s *store) swapped() { // want swapped:"^a send to a channel when its callers hold 2 locks$"
	s.mu.Unlock()
	s.rw.Lock()
	s.ch <- 1 // want `a send to a channel blocks inside the critical section of s.rw from line 198`
	s.rw.Unlock()
	s.mu.Lock()
}

// triedFirst sends after a section that TryLock began, while the lock of its caller is held.
func (s *store) triedFirst() { // want triedFirst:"^a send to a channel$"
	if s.mu.TryLock() {
		s.mu.Unlock()
	}
	s.ch <- 1
}

// lockedOnOnePath sends after a section that one path alone locks, while the lock of its caller is
// held.
func (s *store) lockedOnOnePath(lock bool) { // want lockedOnOnePath:"^a send to a channel$"
	if lock {
		s.mu.Lock()
	}
	if lock {
		s.mu.Unlock()
	}
	s.ch <- 1
}
