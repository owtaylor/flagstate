package util

import (
	"sync/atomic"
	"testing"
)

func expectRepo(t *testing.T, rd *repoDispatcher, expected string) {
	repo := rd.Take()
	defer rd.Release(repo)
	if repo != expected {
		t.Errorf("Expected '%s', got '%s'", expected, repo)
	}
}

func TestRepoDispatcherBasic(t *testing.T) {
	rd := NewRepoDispatcher()

	rd.Add("foo", false)
	expectRepo(t, rd, "foo")

	rd.Add("foo", true)
	rd.Add("bar", true)
	rd.Add("baz", false)
	expectRepo(t, rd, "baz")
	rd.Release(rd.Take())
	rd.Release(rd.Take())

	rd.Add("foo", true)
	repo := rd.Take()
	rd.Add("foo", true)
	rd.Add("bar", true)
	expectRepo(t, rd, "bar")
	rd.Release(repo)
	expectRepo(t, rd, "foo")
}

func TestRepoDispatcherLock(t *testing.T) {
	rd := NewRepoDispatcher()
	ch := make(chan bool)
	var v atomic.Value

	go func() {
		for i := 0; i < 2; i++ {
			repo := rd.Take()
			ch <- true
			// Busy wait until the main thread has locked
			locked := false
			for !locked {
				rd.mutex.Lock()
				locked = rd.locked
				rd.mutex.Unlock()
			}
			v.Store(i + 1)
			rd.Release(repo)
		}
	}()

	rd.Add("foo", false)
	rd.Add("bar", false)
	for i := 0; i < 2; i++ {
		// Wait until a worker is busy
		<-ch
		// This should wait until a worker is released
		rd.Lock()
		processed, _ := v.Load().(int)
		if processed != (i + 1) {
			t.Errorf("On iteration %d, got %d", i, processed)
		}
		rd.Unlock()
	}
}
