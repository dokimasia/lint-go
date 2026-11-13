// Package calls is the package of the cases of lockblock through the calls inside one package.
package calls

import "sync"

// mu is the lock of the cases.
var mu sync.Mutex

// ch is the channel of the cases.
var ch = make(chan int)

// send sends to ch.
func send() { // want send:"a send to a channel"
	ch <- 1
}

// relay calls send.
func relay() { // want relay:"calls.send -> a send to a channel"
	send()
}

// countdown calls itself until n is 0, and then sends.
func countdown(n int) { // want countdown:"calls.send -> a send to a channel"
	if n == 0 {
		send()
		return
	}
	countdown(n - 1)
}

// quiet makes no blocking operation.
func quiet() int {
	return len(ch)
}

// waitOwn waits on cond while its own lock is held, which keeps the lock of its caller.
func waitOwn(own *sync.Mutex, cond *sync.Cond) { // want waitOwn:"^sync.Cond.Wait$"
	own.Lock()
	cond.Wait()
	own.Unlock()
}

// waitCallers waits on cond without a lock of its own, which releases the lock of its caller.
func waitCallers(cond *sync.Cond) { // want waitCallers:"^sync.Cond.Wait when its callers hold 2 locks$"
	cond.Wait()
}

// calling calls each function while mu is locked.
func calling(own *sync.Mutex, cond *sync.Cond, s *store) { // want calling:"calls.relay -> calls.send -> a send to a channel"
	mu.Lock()
	relay()            // want `the call of calls.relay blocks inside the critical section of mu from line 50: calls.relay -> calls.send -> a send to a channel`
	countdown(3)       // want `the call of calls.countdown blocks inside the critical section of mu from line 50: calls.countdown -> calls.send -> a send to a channel`
	quiet()
	waitOwn(own, cond) // want `the call of calls.waitOwn blocks inside the critical section of mu from line 50: calls.waitOwn -> sync.Cond.Wait`
	waitCallers(cond)
	func() { <-ch }()  // want `the call of a function literal of calls.calling blocks inside the critical section of mu from line 50: a function literal of calls.calling -> a receive from a channel`
	s.fit()            // want `the call of \(\*calls.store\).fit blocks inside the critical section of mu from line 50: \(\*calls.store\).fit -> \(\*calls.store\).await -> a receive from a channel`
	mu.Unlock()
}

// store has a lock and a channel.
type store struct {
	mu sync.Mutex
	ch chan int
}

// await receives while the lock mu of its caller is unlocked, and then locks mu again.
func (s *store) await() { // want await:"^a receive from a channel when its callers hold 2 locks$"
	s.mu.Unlock()
	<-s.ch
	s.mu.Lock()
}

// depend calls await without a lock of its own.
func (s *store) depend() { // want depend:`^\(\*calls.store\).await -> a receive from a channel when its callers hold 2 locks$`
	s.await()
}

// fit calls await while mu is locked, so the receive blocks inside a critical section of its caller.
func (s *store) fit() { // want fit:`^\(\*calls.store\).await -> a receive from a channel$`
	s.mu.Lock()
	defer s.mu.Unlock()
	s.await()
}

// awaiting calls await and depend while mu alone is locked, and await while other and mu are
// locked.
func awaiting(s *store, other *sync.Mutex) { // want awaiting:`^\(\*calls.store\).await -> a receive from a channel$`
	s.mu.Lock()
	s.await()
	s.depend()
	s.mu.Unlock()
	other.Lock()
	s.mu.Lock()
	s.await() // want `the call of \(\*calls.store\).await blocks inside the critical section of other from line 93: \(\*calls.store\).await -> a receive from a channel`
	s.mu.Unlock()
	other.Unlock()
}
