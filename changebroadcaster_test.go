package main

import (
	"testing"
	"time"
)

func TestChangeBroadcaster(t *testing.T) {
	cb := NewChangeBroadcaster()

	lastChange := cb.Change()

	if res := cb.LastChange(); res != lastChange {
		t.Errorf("Expected %+v got %+v", lastChange, res)
	}

	newChange := cb.Change()

	// Test the case where data is waiting already
	change := cb.Wait(lastChange)
	if change != newChange {
		t.Errorf("Expected %+v got %+v", newChange, change)

	}
	change, ok := cb.WaitTimeout(lastChange, 0)
	if !ok || change != newChange {
		t.Errorf("Expected (%+v, true), got (%+v, %v)", newChange, change, ok)
	}

	// Test for immediate failure with a zero time
	lastChange = cb.Change()
	_, ok = cb.WaitTimeout(lastChange, 0)
	if ok {
		t.Errorf("Expected no result")
	}

	ch := make(chan Change)

	// Test actual waiting in a separate goroutine
	lastChange = cb.Change()
	go func() {
		ch <- cb.Wait(lastChange)
	}()

	time.Sleep(10 * time.Millisecond)
	newChange = cb.Change()
	change = <-ch
	if change != newChange {
		t.Errorf("Expected %+v, got %+v", newChange, change)
	}

	// Test actual waiting with a timeout that doesn't expire
	lastChange = cb.Change()
	go func() {
		change, ok := cb.WaitTimeout(lastChange, 1000*time.Millisecond)
		if !ok {
			t.Errorf("Expected success")
		}
		ch <- change
	}()

	time.Sleep(10 * time.Millisecond)
	newChange = cb.Change()
	change = <-ch
	if change != newChange {
		t.Errorf("Expected %+v got %+v", newChange, change)
	}

	// Test actual waiting with a timeout that expires
	lastChange = cb.Change()
	go func() {
		change, ok := cb.WaitTimeout(lastChange, 10*time.Millisecond)
		if ok {
			t.Errorf("Expected failure")
		}
		ch <- change
	}()

	time.Sleep(100 * time.Millisecond)
	newChange = cb.Change()
	change = <-ch
}
