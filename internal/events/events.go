// Package events defines the request and reply records that flow through the
// tool, and an in-memory store that publishes them to subscribers.
//
// Every other package reads from or writes to this stream; the screen, the
// headless mode and the export code never talk to the HTTP server directly.
package events

import (
	"sync"
	"time"
)

// Header is one HTTP header line. Order is preserved, names keep their case.
type Header struct {
	Name  string
	Value string
}

// Request is an incoming HTTP request as captured by the server.
type Request struct {
	Time          time.Time
	Method        string
	Target        string // path and query, exactly as sent
	Proto         string
	Host          string
	Remote        string
	Headers       []Header // sorted by name, includes Host
	Body          []byte
	BodyTruncated bool // true when the body was larger than the size limit
}

// Response is the reply that was sent for a request.
type Response struct {
	Status  int
	Headers []Header
	Body    []byte
	Delay   time.Duration
}

// Event is one request and, once available, its reply.
type Event struct {
	ID       int
	Request  Request
	Response *Response // nil until the reply has been sent
	Duration time.Duration
	Done     bool // true once the reply was sent or the client went away
	Aborted  bool // true when the client disconnected before the reply
}

// DefaultCapacity is how many events the store keeps by default.
const DefaultCapacity = 1000

const subscriberBuffer = 1024

// Store keeps the most recent events and publishes changes to subscribers.
// It is safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	cap     int
	nextID  int
	items   []Event
	nextSub int
	subs    map[int]chan Event
}

// NewStore returns a store that keeps at most capacity events.
func NewStore(capacity int) *Store {
	if capacity < 1 {
		capacity = DefaultCapacity
	}
	return &Store{cap: capacity, nextID: 1, subs: map[int]chan Event{}}
}

// Add records a new request and returns its event ID.
func (s *Store) Add(req Request) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ev := Event{ID: s.nextID, Request: req}
	s.nextID++
	s.items = append(s.items, ev)
	if len(s.items) > s.cap {
		copy(s.items, s.items[1:])
		s.items = s.items[:s.cap]
	}
	s.publish(ev)
	return ev.ID
}

// Complete attaches the reply to an event. It does nothing if the event was
// already evicted from the store.
func (s *Store) Complete(id int, resp Response, took time.Duration) {
	s.finish(id, &resp, took, false)
}

// Abort marks an event as finished without a reply (the client went away).
func (s *Store) Abort(id int, took time.Duration) {
	s.finish(id, nil, took, true)
}

func (s *Store) finish(id int, resp *Response, took time.Duration, aborted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.items) - 1; i >= 0; i-- {
		if s.items[i].ID != id {
			continue
		}
		s.items[i].Response = resp
		s.items[i].Duration = took
		s.items[i].Done = true
		s.items[i].Aborted = aborted
		s.publish(s.items[i])
		return
	}
}

// Snapshot returns a copy of the stored events, oldest first.
func (s *Store) Snapshot() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.items))
	copy(out, s.items)
	return out
}

// Subscribe returns a channel of event updates and a function that cancels the
// subscription. Each change (new request, completed reply) is sent as a full
// Event; receivers should replace any earlier copy with the same ID.
//
// A subscriber that falls more than 1024 events behind loses updates instead
// of slowing the server down.
func (s *Store) Subscribe() (<-chan Event, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextSub
	s.nextSub++
	ch := make(chan Event, subscriberBuffer)
	s.subs[id] = ch
	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.subs[id]; ok {
			delete(s.subs, id)
			close(c)
		}
	}
	return ch, cancel
}

// publish must be called with s.mu held.
func (s *Store) publish(ev Event) {
	for _, ch := range s.subs {
		select {
		case ch <- ev:
		default: // slow subscriber: drop rather than block the server
		}
	}
}
