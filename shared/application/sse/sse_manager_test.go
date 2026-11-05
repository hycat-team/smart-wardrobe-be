package sse

import (
	"testing"
	"time"
)

func newTestManager(maxSize int, ttl time.Duration) *SSEManager {
	return &SSEManager{
		channels: make(map[string][]chan Event),
		buffers:  make(map[string]*taskBuffer),
		maxSize:  maxSize,
		ttl:      ttl,
		now:      time.Now,
	}
}

func drain(ch chan Event) []string {
	var got []string
	for {
		select {
		case ev := <-ch:
			got = append(got, string(ev.Data))
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
	if got := <-ch; string(got.Data) != "c" {
		t.Fatalf("live want c, got %s", got.Data)
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

// A replay must carry the SAME id the live subscriber saw. That identity is what
// lets the client discard a replayed frame instead of re-applying it — the
// wardrobe badge used to flicker because a reconnect replayed an old
// "processing" event after "failed" had already been shown.
func TestSSEManager_ReplayKeepsOriginalIDs(t *testing.T) {
	m := newTestManager(5, time.Minute)

	live := m.Subscribe("t1")
	m.Publish("t1", []byte("processing"))
	first := <-live
	m.Publish("t1", []byte("failed"))
	second := <-live

	replayed := drainEvents(m.Subscribe("t1"))
	if len(replayed) != 2 {
		t.Fatalf("want 2 replayed events, got %d", len(replayed))
	}
	if replayed[0].ID != first.ID || replayed[1].ID != second.ID {
		t.Fatalf("replay ids = %d,%d want %d,%d",
			replayed[0].ID, replayed[1].ID, first.ID, second.ID)
	}
}

// Ids must be globally increasing so a client comparing "last seen id" cannot be
// fooled by two tasks interleaving, or by a second manager instance started
// later (a deploy) handing out ids from the beginning again.
func TestSSEManager_IDsIncreaseAcrossTasks(t *testing.T) {
	m := newTestManager(5, time.Minute)
	a := m.Subscribe("t1")
	b := m.Subscribe("t2")

	m.Publish("t1", []byte("a"))
	m.Publish("t2", []byte("b"))

	first := <-a
	second := <-b
	if second.ID <= first.ID {
		t.Fatalf("ids must increase globally, got %d then %d", first.ID, second.ID)
	}

	later := NewSSEManager(nil)
	ch := later.Subscribe("t1")
	later.Publish("t1", []byte("c"))
	if got := (<-ch).ID; got <= second.ID {
		t.Fatalf("a manager started later must hand out larger ids, got %d <= %d", got, second.ID)
	}
}

func drainEvents(ch chan Event) []Event {
	var got []Event
	for {
		select {
		case ev := <-ch:
			got = append(got, ev)
		default:
			return got
		}
	}
}
