// Package server is the HTTP listener. It records every request as an event,
// asks the responder for the reply, waits for the delay, and sends the reply.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/responder"
)

// readTimeout bounds how long a client may take to send its request. It is a
// variable so tests can shorten it.
var readTimeout = 30 * time.Second

// Options configures a Server.
type Options struct {
	Addr    string // host:port to listen on
	MaxBody int64  // largest request body kept, in bytes
	CORS    bool   // answer preflight requests and add CORS headers
}

// Server implements http.Handler and can also run its own listener.
type Server struct {
	opts  Options
	store *events.Store
	resp  *responder.Responder
	ln    net.Listener
}

// New returns a Server. Call Listen and then Serve to run it.
func New(opts Options, store *events.Store, resp *responder.Responder) *Server {
	if opts.MaxBody < 1 {
		opts.MaxBody = 10 << 20
	}
	return &Server{opts: opts, store: store, resp: resp}
}

// Listen binds the address now, so problems such as a busy port are reported
// before any screen is drawn.
func (s *Server) Listen() (net.Addr, error) {
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", s.opts.Addr)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	return ln.Addr(), nil
}

// IsAddrInUse reports whether err means the port is already taken.
func IsAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

// Serve runs until ctx is cancelled, then shuts down gracefully.
func (s *Server) Serve(ctx context.Context) error {
	if s.ln == nil {
		return errors.New("server: Listen must be called before Serve")
	}
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       readTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          log.New(io.Discard, "", 0), // never write over the screen
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(s.ln) }()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
		}
		<-errc
		return nil
	}
}

// ServeHTTP handles one request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	body, truncated := readBody(r.Body, s.opts.MaxBody)
	id := s.store.Add(events.Request{
		Time:          start,
		Method:        r.Method,
		Target:        r.RequestURI,
		Proto:         r.Proto,
		Host:          r.Host,
		Remote:        r.RemoteAddr,
		Headers:       collectHeaders(r),
		Body:          body,
		BodyTruncated: truncated,
	})

	if s.opts.CORS && isPreflight(r) {
		resp := events.Response{Status: http.StatusNoContent, Headers: corsHeaders(r, true)}
		write(w, resp)
		s.store.Complete(id, resp, time.Since(start))
		return
	}

	resp := s.resp.Render()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(resp.Delay + 30*time.Second))
	if resp.Delay > 0 {
		timer := time.NewTimer(resp.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-r.Context().Done():
			s.store.Abort(id, time.Since(start))
			return
		}
	}
	if s.opts.CORS {
		resp.Headers = mergeMissing(resp.Headers, corsHeaders(r, false))
	}
	write(w, resp)
	s.store.Complete(id, resp, time.Since(start))
}

func readBody(body io.Reader, max int64) ([]byte, bool) {
	if body == nil {
		return nil, false
	}
	data, _ := io.ReadAll(io.LimitReader(body, max+1)) // a partial body is still worth showing
	if int64(len(data)) > max {
		return data[:max], true
	}
	return data, false
}

func collectHeaders(r *http.Request) []events.Header {
	hs := []events.Header{{Name: "Host", Value: r.Host}}
	for name, values := range r.Header {
		for _, v := range values {
			hs = append(hs, events.Header{Name: name, Value: v})
		}
	}
	if len(r.TransferEncoding) > 0 {
		hs = append(hs, events.Header{Name: "Transfer-Encoding", Value: strings.Join(r.TransferEncoding, ", ")})
	}
	sort.SliceStable(hs, func(i, j int) bool {
		return strings.ToLower(hs[i].Name) < strings.ToLower(hs[j].Name)
	})
	return hs
}

func write(w http.ResponseWriter, resp events.Response) {
	h := w.Header()
	for _, hh := range resp.Headers {
		h.Add(hh.Name, hh.Value)
	}
	w.WriteHeader(resp.Status)
	if len(resp.Body) > 0 && bodyAllowed(resp.Status) {
		_, _ = w.Write(resp.Body)
	}
}

func bodyAllowed(status int) bool {
	informational := status >= 100 && status < 200
	return !informational && status != http.StatusNoContent && status != http.StatusNotModified
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func corsHeaders(r *http.Request, preflight bool) []events.Header {
	origin := r.Header.Get("Origin")
	allow := "*"
	var hs []events.Header
	if origin != "" {
		allow = origin
		hs = append(hs, events.Header{Name: "Vary", Value: "Origin"})
	}
	hs = append(hs, events.Header{Name: "Access-Control-Allow-Origin", Value: allow})
	if preflight {
		methods := r.Header.Get("Access-Control-Request-Method")
		if methods == "" {
			methods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
		}
		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		if reqHeaders == "" {
			reqHeaders = "*"
		}
		hs = append(hs,
			events.Header{Name: "Access-Control-Allow-Methods", Value: methods},
			events.Header{Name: "Access-Control-Allow-Headers", Value: reqHeaders},
			events.Header{Name: "Access-Control-Max-Age", Value: "600"},
		)
	}
	return hs
}

// mergeMissing appends extra headers whose names are not already present.
func mergeMissing(base, extra []events.Header) []events.Header {
	have := map[string]bool{}
	for _, h := range base {
		have[strings.ToLower(h.Name)] = true
	}
	out := append([]events.Header(nil), base...)
	for _, h := range extra {
		if !have[strings.ToLower(h.Name)] {
			out = append(out, h)
		}
	}
	return out
}

// Describe returns a short human description of the listener, for messages.
func Describe(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
}
