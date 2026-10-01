package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/responder"
)

func setup(t *testing.T, opts Options, spec responder.Spec) (*httptest.Server, *events.Store) {
	t.Helper()
	store := events.NewStore(50)
	srv := New(opts, store, responder.New(spec, responder.EnvPolicy{}))
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, store
}

func TestCapturesRequestAndSendsReply(t *testing.T) {
	spec := responder.Spec{
		Status:  201,
		Headers: []events.Header{{Name: "X-Reply", Value: "yes"}},
		Body:    "created",
	}
	ts, store := setup(t, Options{MaxBody: 1024}, spec)

	req, _ := http.NewRequest("POST", ts.URL+"/hook?x=1", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zeta", "last")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 201 || string(body) != "created" || resp.Header.Get("X-Reply") != "yes" {
		t.Fatalf("unexpected reply: %d %q %v", resp.StatusCode, body, resp.Header)
	}
	evs := store.Snapshot()
	if len(evs) != 1 || !evs[0].Done {
		t.Fatalf("expected one completed event, got %+v", evs)
	}
	r := evs[0].Request
	if r.Method != "POST" || r.Target != "/hook?x=1" || string(r.Body) != `{"a":1}` || r.BodyTruncated {
		t.Fatalf("unexpected request: %+v", r)
	}
	for i := 1; i < len(r.Headers); i++ {
		if strings.ToLower(r.Headers[i-1].Name) > strings.ToLower(r.Headers[i].Name) {
			t.Fatalf("headers not sorted: %+v", r.Headers)
		}
	}
	if last := r.Headers[len(r.Headers)-1]; last.Name != "Zeta" {
		t.Fatalf("last header should be Zeta, got %+v", last)
	}
	if evs[0].Response.Status != 201 {
		t.Fatalf("response not recorded: %+v", evs[0].Response)
	}
}

func TestBodyIsTruncatedAtLimit(t *testing.T) {
	ts, store := setup(t, Options{MaxBody: 5}, responder.Default())
	resp, err := http.Post(ts.URL, "text/plain", strings.NewReader("0123456789"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	r := store.Snapshot()[0].Request
	if string(r.Body) != "01234" || !r.BodyTruncated {
		t.Fatalf("body=%q truncated=%v", r.Body, r.BodyTruncated)
	}
}

func TestCORSPreflight(t *testing.T) {
	ts, store := setup(t, Options{CORS: true}, responder.Default())
	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "PUT")
	req.Header.Set("Access-Control-Request-Headers", "x-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 ||
		resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" ||
		resp.Header.Get("Access-Control-Allow-Methods") != "PUT" ||
		resp.Header.Get("Access-Control-Allow-Headers") != "x-token" {
		t.Fatalf("bad preflight reply: %d %v", resp.StatusCode, resp.Header)
	}
	if len(store.Snapshot()) != 1 {
		t.Fatal("preflight must also be recorded")
	}
}

func TestCORSHeadersOnNormalReplyAndOffByDefault(t *testing.T) {
	on, _ := setup(t, Options{CORS: true}, responder.Default())
	off, _ := setup(t, Options{CORS: false}, responder.Default())
	for url, want := range map[string]string{on.URL: "*", off.URL: ""} {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != want {
			t.Fatalf("%s: allow-origin = %q, want %q", url, got, want)
		}
	}
}

func TestDelayAndClientAbort(t *testing.T) {
	spec := responder.Default()
	spec.Delay = 400 * time.Millisecond
	ts, store := setup(t, Options{}, spec)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	if r, err := http.DefaultClient.Do(req); err == nil {
		r.Body.Close()
		t.Fatal("expected the client to time out")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		evs := store.Snapshot()
		if len(evs) == 1 && evs[0].Aborted {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("event was not marked aborted: %+v", store.Snapshot())
}

func TestNoBodyFor204(t *testing.T) {
	ts, _ := setup(t, Options{}, responder.Spec{Status: 204, Body: "ignored"})
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 204 || len(body) != 0 {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
}

func TestListenServeAndShutdown(t *testing.T) {
	store := events.NewStore(5)
	srv := New(Options{Addr: "127.0.0.1:0"}, store, responder.New(responder.Default(), responder.EnvPolicy{}))
	addr, err := srv.Listen()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx) }()

	resp, err := http.Get("http://" + addr.String())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
}

func TestPortInUseIsDetected(t *testing.T) {
	first := New(Options{Addr: "127.0.0.1:0"}, events.NewStore(1), nil)
	addr, err := first.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer first.ln.Close()
	second := New(Options{Addr: addr.String()}, events.NewStore(1), nil)
	if _, err := second.Listen(); !IsAddrInUse(err) {
		t.Fatalf("expected an address-in-use error, got %v", err)
	}
}

// A reply delayed for longer than the server's read timeout must still be sent:
// the read timeout is for receiving the request, not for waiting to reply.
func TestDelayLongerThanReadTimeoutStillReplies(t *testing.T) {
	old := readTimeout
	readTimeout = 300 * time.Millisecond
	t.Cleanup(func() { readTimeout = old })

	store := events.NewStore(5)
	spec := responder.Default()
	spec.Delay = 900 * time.Millisecond
	srv := New(Options{Addr: "127.0.0.1:0"}, store, responder.New(spec, responder.EnvPolicy{}))
	addr, err := srv.Listen()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Serve(ctx) }()

	resp, err := http.Get("http://" + addr.String())
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	evs := store.Snapshot()
	if len(evs) != 1 || evs[0].Aborted || evs[0].Response == nil {
		t.Fatalf("event should be completed, not aborted: %+v", evs)
	}
}
