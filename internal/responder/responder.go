// Package responder decides what reply the server sends.
//
// A Spec is the reply the user designed (status, headers, body, delay). The
// Responder holds the current Spec, lets the screen replace it at any time, and
// renders it into an events.Response, expanding ${VAR} placeholders safely.
package responder

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tejjasdev/portbell/internal/brand"
	"github.com/tejjasdev/portbell/internal/events"
)

// Limits for a reply.
const (
	MinStatus = 200 // 1xx codes are interim responses; a client would see a different final status
	MaxStatus = 599
	MaxDelay  = 10 * time.Minute
)

// Spec is a reply as designed by the user.
type Spec struct {
	Status  int
	Headers []events.Header
	Body    string
	Delay   time.Duration
}

// Default returns the reply used when nothing else is configured.
func Default() Spec {
	return Spec{
		Status:  200,
		Headers: []events.Header{{Name: "Content-Type", Value: "text/plain; charset=utf-8"}},
		Body:    "Hello from " + brand.Name + "\n",
	}
}

// Validate reports the first problem that would make the Spec unsafe or
// impossible to send.
func (s Spec) Validate() error {
	if s.Status < MinStatus || s.Status > MaxStatus {
		return fmt.Errorf("status %d is out of range (%d-%d)", s.Status, MinStatus, MaxStatus)
	}
	if s.Delay < 0 || s.Delay > MaxDelay {
		return fmt.Errorf("delay %s is out of range (0-%s)", s.Delay, MaxDelay)
	}
	for _, h := range s.Headers {
		if !validHeaderName(h.Name) {
			return fmt.Errorf("invalid header name %q", h.Name)
		}
		if !validHeaderValue(h.Value) {
			return fmt.Errorf("invalid value for header %q (control characters are not allowed)", h.Name)
		}
	}
	return nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
		if !ok {
			return false
		}
	}
	return true
}

func validHeaderValue(v string) bool {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\t' {
			continue
		}
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

// EnvPolicy says which environment variables ${VAR} may read.
//
// A variable is readable if its name starts with Prefix or appears in Allow.
// This keeps a shared reply file from leaking a secret (for example
// AWS_SECRET_ACCESS_KEY) to whoever calls the server.
type EnvPolicy struct {
	Prefix string
	Allow  []string
	Lookup func(string) (string, bool) // defaults to os.LookupEnv
}

var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Expand replaces ${NAME} with the value of an allowed environment variable.
// Unknown, unset or disallowed names are left as they are, so the problem is
// visible in the reply instead of silently becoming empty text.
func (p EnvPolicy) Expand(s string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	lookup := p.Lookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	return placeholder.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		if !p.allowed(name) {
			return m
		}
		if v, ok := lookup(name); ok {
			return v
		}
		return m
	})
}

func (p EnvPolicy) allowed(name string) bool {
	if p.Prefix != "" && strings.HasPrefix(name, p.Prefix) {
		return true
	}
	for _, a := range p.Allow {
		if a == name {
			return true
		}
	}
	return false
}

// Responder holds the current reply and is safe for concurrent use.
type Responder struct {
	mu   sync.RWMutex
	spec Spec
	env  EnvPolicy
}

// New returns a Responder using spec. The Spec must already be valid.
func New(spec Spec, env EnvPolicy) *Responder {
	return &Responder{spec: spec, env: env}
}

// Get returns the current Spec.
func (r *Responder) Get() Spec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.spec
}

// Set replaces the current Spec after validating it.
func (r *Responder) Set(s Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	r.spec = s
	r.mu.Unlock()
	return nil
}

func sanitizeHeaderValue(v string) string {
	if validHeaderValue(v) {
		return v
	}
	var b strings.Builder
	b.Grow(len(v))
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\t' || (c >= 0x20 && c != 0x7f) {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// Render returns the reply to send now, with placeholders expanded.
func (r *Responder) Render() events.Response {
	s := r.Get()
	headers := make([]events.Header, len(s.Headers))
	for i, h := range s.Headers {
		headers[i] = events.Header{Name: h.Name, Value: sanitizeHeaderValue(r.env.Expand(h.Value))}
	}
	return events.Response{
		Status:  s.Status,
		Headers: headers,
		Body:    []byte(r.env.Expand(s.Body)),
		Delay:   s.Delay,
	}
}
