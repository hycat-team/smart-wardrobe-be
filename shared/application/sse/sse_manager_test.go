package sse

import (
	"testing"
	"time"
)

func newTestManager(maxSize int, ttl time.Duration) *SSEManager {
	return &SSEManager{
		channels: make(map[string][]chan []byte),
		buffers:  make(map[string]*taskBuffer),
		maxSize:  maxSize,
		ttl:      ttl,
		now:      time.Now,
	}
}

func drain(ch chan []byte) []string {
	var got []string
	for {
		select {
		case ev := <-ch:
			got = append(got, string(ev))
		default:
			return got
		}
	}
}

func TestSSEManager_ReplayFIFOThenLive(t *testing.T) {
	m := newTestManager(5, time.Minute)
	m.Publish("t1", []byte("a"))
	m.Publish("t1", []byte("b"))

	ch := m.Subscribe("t1")
	if got := drain(ch); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("replay want [a b], got %v", got)
	}

	m.Publish("t1", []byte("c"))
	if got := <-ch; string(got) != "c" {
		t.Fatalf("live want c, got %s", got)
	}
}

func TestSSEManager_BufferDropsOldestWhenFull(t *testing.T) {
	m := newTestManager(2, time.Minute)
	m.Publish("t1", []byte("a"))
	m.Publish("t1", []byte("b"))
	m.Publish("t1", []byte("c"))

	ch := m.Subscribe("t1")
	got := drain(ch)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("want [b c], got %v", got)
	}
}

func TestSSEManager_TTLExpiry(t *testing.T) {
	base := time.Now()
	m := newTestManager(5, time.Minute)
	m.now = func() time.Time { return base }
	m.Publish("t1", []byte("a"))

	m.now = func() time.Time { return base.Add(2 * time.Minute) }
	ch := m.Subscribe("t1")
	if got := drain(ch); len(got) != 0 {
		t.Fatalf("expired buffer should replay nothing, got %v", got)
	}
}

func TestSSEManager_BufferKeptAfterUnsubscribe(t *testing.T) {
	m := newTestManager(5, time.Minute)
	ch := m.Subscribe("t1")
	m.Publish("t1", []byte("a"))
	m.Unsubscribe("t1", ch)

	m.Publish("t1", []byte("b"))
	ch2 := m.Subscribe("t1")
	if got := drain(ch2); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("want [a b] after reconnect, got %v", got)
	}
}
