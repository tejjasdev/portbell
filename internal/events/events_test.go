package events

import (
	"testing"
	"time"
)

func TestStoreAddCompleteAndSnapshot(t *testing.T) {
	s := NewStore(10)
	id := s.Add(Request{Method: "GET", Target: "/a"})
	s.Complete(id, Response{Status: 200}, 5*time.Millisecond)

	got := s.Snapshot()
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if !got[0].Done || got[0].Response == nil || got[0].Response.Status != 200 {
		t.Fatalf("event not completed: %+v", got[0])
	}
	if got[0].Duration != 5*time.Millisecond {
		t.Fatalf("duration = %v", got[0].Duration)
	}
}

func TestStoreKeepsOnlyCapacity(t *testing.T) {
	s := NewStore(3)
	for i := 0; i < 5; i++ {
		s.Add(Request{Target: "/x"})
	}
	got := s.Snapshot()
	if len(got) != 3 {
		t.Fatalf("want 3 events, got %d", len(got))
	}
	if got[0].ID != 3 || got[2].ID != 5 {
		t.Fatalf("wrong events kept: first=%d last=%d", got[0].ID, got[2].ID)
	}
}

func TestCompleteOnEvictedEventIsIgnored(t *testing.T) {
	s := NewStore(1)
	first := s.Add(Request{})
	s.Add(Request{})
	s.Complete(first, Response{Status: 200}, 0) // must not panic or add anything
	if got := s.Snapshot(); len(got) != 1 || got[0].Done {
		t.Fatalf("unexpected state: %+v", got)
	}
}

func TestSubscribeReceivesUpdatesAndCancel(t *testing.T) {
	s := NewStore(5)
	ch, cancel := s.Subscribe()
	id := s.Add(Request{Method: "POST"})
	s.Complete(id, Response{Status: 201}, 0)

	first := <-ch
	second := <-ch
	if first.Done || !second.Done || second.Response.Status != 201 {
		t.Fatalf("unexpected updates: %+v / %+v", first, second)
	}
	cancel()
	cancel() // cancelling twice is safe
	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after cancel")
	}
}

func TestAbort(t *testing.T) {
	s := NewStore(5)
	id := s.Add(Request{})
	s.Abort(id, time.Second)
	got := s.Snapshot()[0]
	if !got.Done || !got.Aborted || got.Response != nil {
		t.Fatalf("unexpected: %+v", got)
	}
}
