package util

import (
	"sync"
	"time"
)

type Change struct {
	serial uint64
}

type ChangeBroadcaster struct {
	serial  uint64
	mutex   sync.Mutex
	waiters []chan Change
}

func NewChangeBroadcaster() *ChangeBroadcaster {
	return &ChangeBroadcaster{}
}

func (cb *ChangeBroadcaster) Change() Change {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.serial++
	change := Change{
		serial: cb.serial,
	}

	for i := range cb.waiters {
		select {
		case cb.waiters[i] <- change:
		default:
		}
	}

	return change
}

func (cb *ChangeBroadcaster) LastChange() Change {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return Change{
		serial: cb.serial,
	}
}

func (cb *ChangeBroadcaster) addWaiter(ch chan Change) {
	cb.waiters = append(cb.waiters, ch)
}

func (cb *ChangeBroadcaster) removeWaiter(ch chan Change) {
	cb.mutex.Lock()
	for i := range cb.waiters {
		if cb.waiters[i] == ch {
			cb.waiters[i] = cb.waiters[len(cb.waiters)-1]
			cb.waiters = cb.waiters[:len(cb.waiters)-1]
			break
		}
	}
	defer cb.mutex.Unlock()
}

func (cb *ChangeBroadcaster) Wait(change Change) Change {
	cb.mutex.Lock()
	if cb.serial > change.serial {
		defer cb.mutex.Unlock()
		return Change{
			serial: cb.serial,
		}
	}

	ch := make(chan Change)
	cb.addWaiter(ch)
	defer cb.removeWaiter(ch)
	cb.mutex.Unlock()

	return <-ch
}

func (cb *ChangeBroadcaster) WaitTimeout(change Change, timeout time.Duration) (Change, bool) {
	cb.mutex.Lock()
	if cb.serial > change.serial {
		defer cb.mutex.Unlock()
		return Change{
			serial: cb.serial,
		}, true
	}
	if timeout <= 0 {
		cb.mutex.Unlock()
		return Change{}, false
	}

	ch := make(chan Change)
	cb.addWaiter(ch)
	defer cb.removeWaiter(ch)
	cb.mutex.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case change := <-ch:
		return change, true
	case <-timer.C:
		return Change{}, false
	}
}
